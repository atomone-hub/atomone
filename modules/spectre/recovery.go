package spectre

import (
	"bytes"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

// recoverClient replaces the frozen subject client's state with the active
// substitute's, unfreezing it under its own identifier. Governance
// authorizes the recovery through MsgRecoverClient; this function enforces
// that the substitute tracks the same Ethereum bridge, so only the progress
// (latest slot, execution block number, consensus states) is taken over,
// never the chain parameters.
func (cs *ClientState) recoverClient(
	cdc codec.BinaryCodec,
	subjectStore, substituteStore storetypes.KVStore,
	substitute *ClientState,
) error {
	if !cs.Frozen {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "client is not frozen: nothing to recover")
	}
	if substitute.LatestSlot <= cs.LatestSlot {
		return errorsmod.Wrapf(clienttypes.ErrInvalidHeight,
			"substitute latest slot %d does not advance subject latest slot %d", substitute.LatestSlot, cs.LatestSlot)
	}
	if !isMatchingClientState(cs, substitute) {
		return errorsmod.Wrap(clienttypes.ErrInvalidSubstitute, "subject and substitute client states differ in chain parameters")
	}

	// The substitute's latest state must exist; recovery without it would
	// leave the client unable to verify its next update.
	latest := substitute.GetLatestHeight()
	if _, found := GetConsensusState(substituteStore, cdc, latest); !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "substitute has no consensus state at its latest slot")
	}

	// The subject's own consensus states are dropped, not kept: the freeze
	// means two thirds of a committee signed conflicting roots, so the
	// subject's pre-freeze states may come from the conflicting branch and
	// would stay provable for the pruning window after recovery. Only the
	// substitute's states, verified by a healthy client, remain.
	prefix := []byte(host.KeyConsensusStatePrefix + "/")
	iterator := subjectStore.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	for ; iterator.Valid(); iterator.Next() {
		subjectStore.Delete(iterator.Key())
	}
	iterator.Close()

	iterator = substituteStore.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	for ; iterator.Valid(); iterator.Next() {
		subjectStore.Set(iterator.Key(), iterator.Value())
	}
	iterator.Close()

	recovered := cs.Clone()
	recovered.Frozen = false
	recovered.LatestSlot = substitute.LatestSlot
	recovered.LatestExecutionBlockNumber = substitute.LatestExecutionBlockNumber
	setClientState(subjectStore, cdc, recovered)
	return nil
}

// isMatchingClientState reports whether subject and substitute track the same
// chain and bridge: every field except the mutable progress (frozen flag,
// latest slot, latest execution block number) must be equal.
func isMatchingClientState(subject, substitute *ClientState) bool {
	forksEqual := func(a, b *ForkParameters) bool {
		forkEqual := func(a, b Fork) bool {
			return a.Epoch == b.Epoch && bytes.Equal(a.Version, b.Version)
		}
		return bytes.Equal(a.GenesisForkVersion, b.GenesisForkVersion) &&
			forkEqual(a.Altair, b.Altair) &&
			forkEqual(a.Bellatrix, b.Bellatrix) &&
			forkEqual(a.Capella, b.Capella) &&
			forkEqual(a.Deneb, b.Deneb) &&
			forkEqual(a.Electra, b.Electra) &&
			forkEqual(a.Fulu, b.Fulu) &&
			forkEqual(a.Gloas, b.Gloas)
	}

	return subject.ChainId == substitute.ChainId &&
		subject.GenesisSlot == substitute.GenesisSlot &&
		bytes.Equal(subject.GenesisValidatorsRoot, substitute.GenesisValidatorsRoot) &&
		subject.SecondsPerSlot == substitute.SecondsPerSlot &&
		subject.SlotsPerEpoch == substitute.SlotsPerEpoch &&
		subject.EpochsPerSyncCommitteePeriod == substitute.EpochsPerSyncCommitteePeriod &&
		subject.SyncCommitteeSize == substitute.SyncCommitteeSize &&
		subject.MinSyncCommitteeParticipants == substitute.MinSyncCommitteeParticipants &&
		bytes.Equal(subject.IbcContractAddress, substitute.IbcContractAddress) &&
		bytes.Equal(subject.IbcCommitmentSlot, substitute.IbcCommitmentSlot) &&
		forksEqual(&subject.ForkParameters, &substitute.ForkParameters)
}
