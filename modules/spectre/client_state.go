package spectre

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ exported.ClientState = (*ClientState)(nil)

// ClientType returns the spectre client type.
func (ClientState) ClientType() string {
	return ModuleName
}

// GetLatestHeight returns the latest finalized slot as the client height.
// The revision number is always zero: the Ethereum chain ID lives in the
// client state and Ethereum has no IBC chain-revision concept.
func (cs *ClientState) GetLatestHeight() exported.Height {
	return clienttypes.NewHeight(0, cs.LatestSlot)
}

// GetTimestampAtHeight returns the timestamp in nanoseconds of the consensus
// state at the given height.
func (cs *ClientState) GetTimestampAtHeight(
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	consState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return 0, errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "height (%s)", height)
	}
	return consState.GetTimestamp(), nil
}

// status returns the client status: Frozen when frozen by misbehaviour,
// Active otherwise. The reference implementation does not report expiry.
func (cs *ClientState) status(_ sdk.Context, _ storetypes.KVStore, _ codec.BinaryCodec) exported.Status {
	if cs.Frozen {
		return exported.Frozen
	}
	return exported.Active
}

// Validate performs basic validation of the client state fields.
func (cs *ClientState) Validate() error {
	if cs.ChainId == 0 {
		return errorsmod.Wrap(ErrInvalidClientState, "chain id must be non-zero")
	}
	if len(cs.GenesisValidatorsRoot) != 32 {
		return errorsmod.Wrap(ErrInvalidClientState, "genesis validators root must be 32 bytes")
	}
	if cs.SecondsPerSlot == 0 {
		return errorsmod.Wrap(ErrInvalidClientState, "seconds per slot must be non-zero")
	}
	if cs.SlotsPerEpoch == 0 {
		return errorsmod.Wrap(ErrInvalidClientState, "slots per epoch must be non-zero")
	}
	if cs.EpochsPerSyncCommitteePeriod == 0 {
		return errorsmod.Wrap(ErrInvalidClientState, "epochs per sync committee period must be non-zero")
	}
	if cs.SyncCommitteeSize == 0 {
		return errorsmod.Wrap(ErrInvalidClientState, "sync committee size must be non-zero")
	}
	if cs.MinSyncCommitteeParticipants == 0 || cs.MinSyncCommitteeParticipants > cs.SyncCommitteeSize {
		return errorsmod.Wrap(ErrInvalidClientState, "minimum sync committee participants must be in (0, sync committee size]")
	}
	if cs.LatestSlot == 0 {
		return errorsmod.Wrap(ErrInvalidClientState, "latest slot must be non-zero")
	}
	if cs.LatestSlot < cs.GenesisSlot {
		return errorsmod.Wrap(ErrInvalidClientState, "latest slot precedes genesis slot")
	}
	if len(cs.IbcContractAddress) != 20 {
		return errorsmod.Wrap(ErrInvalidClientState, "IBC contract address must be 20 bytes")
	}
	if len(cs.IbcCommitmentSlot) != 32 {
		return errorsmod.Wrap(ErrInvalidClientState, "IBC commitment slot must be 32 bytes")
	}
	if err := validateForkParameters(&cs.ForkParameters); err != nil {
		return err
	}
	// The latest slot must be in a fork supported by this light client.
	return cs.verifySupportedForkAtEpoch(cs.computeEpochAtSlot(cs.LatestSlot))
}

// validateForkParameters validates the fork schedule.
func validateForkParameters(fp *ForkParameters) error {
	versions := []struct {
		name    string
		version []byte
	}{
		{"genesis", fp.GenesisForkVersion},
		{"altair", fp.Altair.Version},
		{"bellatrix", fp.Bellatrix.Version},
		{"capella", fp.Capella.Version},
		{"deneb", fp.Deneb.Version},
		{"electra", fp.Electra.Version},
		{"fulu", fp.Fulu.Version},
	}
	for _, v := range versions {
		if len(v.version) != 4 {
			return errorsmod.Wrapf(ErrInvalidClientState, "%s fork version must be 4 bytes", v.name)
		}
	}

	// Fork epochs must be monotonically non-decreasing.
	epochs := []uint64{
		fp.Altair.Epoch, fp.Bellatrix.Epoch, fp.Capella.Epoch,
		fp.Deneb.Epoch, fp.Electra.Epoch, fp.Fulu.Epoch,
	}
	for i := 1; i < len(epochs); i++ {
		if epochs[i] < epochs[i-1] {
			return errorsmod.Wrap(ErrInvalidClientState, "fork epochs must be non-decreasing")
		}
	}
	// A Fulu fork scheduled at or before Electra would make computeForkVersion
	// select Fulu for Electra epochs.
	const notScheduled = ^uint64(0)
	if fp.Fulu.Epoch != notScheduled && fp.Electra.Epoch != notScheduled && fp.Fulu.Epoch <= fp.Electra.Epoch {
		return errorsmod.Wrapf(ErrInvalidClientState, "fulu epoch %d must be after electra epoch %d",
			fp.Fulu.Epoch, fp.Electra.Epoch)
	}
	return nil
}

// initialize stores the initial client and consensus states.
func (cs *ClientState) initialize(
	_ sdk.Context,
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	consState exported.ConsensusState,
) error {
	consensusState, ok := consState.(*ConsensusState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type %T, got %T",
			&ConsensusState{}, consState)
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	if cs.LatestSlot != consensusState.Slot {
		return errorsmod.Wrapf(ErrClientAndConsensusMismatch,
			"client latest slot %d does not match consensus slot %d", cs.LatestSlot, consensusState.Slot)
	}

	setClientState(clientStore, cdc, cs)
	setConsensusState(clientStore, cdc, consensusState, cs.GetLatestHeight())
	return nil
}

// verifyClientMessage verifies a Header or Misbehaviour client message
// without writing any state.
func (cs *ClientState) verifyClientMessage(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	clientMsg exported.ClientMessage,
) error {
	if cs.Frozen {
		return errorsmod.Wrap(ErrFrozen, "client is frozen")
	}

	switch msg := clientMsg.(type) {
	case *Header:
		consState, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, msg.TrustedSlot))
		if !found {
			return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "trusted slot %d", msg.TrustedSlot)
		}
		return cs.verifyHeader(consState, uint64(ctx.BlockTime().Unix()), msg)
	case *Misbehaviour:
		consState, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, msg.TrustedSlot))
		if !found {
			return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "trusted slot %d", msg.TrustedSlot)
		}
		return cs.verifyMisbehaviour(consState, uint64(ctx.BlockTime().Unix()), msg)
	default:
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected type %T or %T, got %T",
			&Header{}, &Misbehaviour{}, clientMsg)
	}
}

// checkForMisbehaviour reports whether the client message is a valid
// misbehaviour report.
func (cs *ClientState) checkForMisbehaviour(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	clientMsg exported.ClientMessage,
) bool {
	if cs.Frozen {
		return false
	}
	mis, ok := clientMsg.(*Misbehaviour)
	if !ok {
		return false
	}
	consState, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, mis.TrustedSlot))
	if !found {
		return false
	}
	return cs.verifyMisbehaviour(consState, uint64(ctx.BlockTime().Unix()), mis) == nil
}

// updateState applies a verified header, storing the new consensus state and
// updating the client state when the finalized slot advanced.
func (cs *ClientState) updateState(
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	clientMsg exported.ClientMessage,
) []exported.Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		panic(errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected type %T, got %T", &Header{}, clientMsg))
	}

	// The update applies against the consensus state at the client's latest
	// slot; historical updates are not supported.
	consState, found := GetConsensusState(clientStore, cdc, cs.GetLatestHeight())
	if !found {
		panic(errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "latest slot %d", cs.LatestSlot))
	}

	newConsState, newClientState, err := cs.updateConsensusState(consState, header)
	if err != nil {
		panic(err)
	}

	height := clienttypes.NewHeight(0, newConsState.Slot)
	setConsensusState(clientStore, cdc, newConsState, height)
	if newClientState != nil {
		setClientState(clientStore, cdc, newClientState)
	}
	return []exported.Height{height}
}

// updateStateOnMisbehaviour freezes the client.
func (cs *ClientState) updateStateOnMisbehaviour(
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	_ exported.ClientMessage,
) {
	if cs.Frozen {
		return
	}
	frozen := cs.Clone()
	frozen.Frozen = true
	setClientState(clientStore, cdc, frozen)
}

// verifyMembership verifies a storage membership proof of value at path at
// the given height against the IBC router contract on Ethereum.
func (cs *ClientState) verifyMembership(
	_ sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
	value []byte,
) error {
	if delayTimePeriod != 0 || delayBlockPeriod != 0 {
		return errorsmod.Wrap(ErrUnsupportedOperation, "non-zero delay periods are not supported")
	}
	if err := validateHeight(height); err != nil {
		return err
	}
	if cs.Frozen {
		return errorsmod.Wrap(ErrFrozen, "client is frozen")
	}
	if cs.LatestSlot < height.GetRevisionHeight() {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidHeight,
			"client state height %d is below proof height %d", cs.LatestSlot, height.GetRevisionHeight())
	}

	var membershipProof MembershipProof
	if err := cdc.Unmarshal(proof, &membershipProof); err != nil {
		return errorsmod.Wrap(ErrInvalidProof, "failed to unmarshal membership proof")
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	consState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "no consensus state at proof height")
	}

	return verifyStorageMembership(cs, consState, &membershipProof, merklePath.KeyPath, value)
}

// verifyNonMembership verifies a storage non-membership proof at path at
// the given height.
func (cs *ClientState) verifyNonMembership(
	_ sdk.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path exported.Path,
) error {
	if delayTimePeriod != 0 || delayBlockPeriod != 0 {
		return errorsmod.Wrap(ErrUnsupportedOperation, "non-zero delay periods are not supported")
	}
	if err := validateHeight(height); err != nil {
		return err
	}
	if cs.Frozen {
		return errorsmod.Wrap(ErrFrozen, "client is frozen")
	}
	if cs.LatestSlot < height.GetRevisionHeight() {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidHeight,
			"client state height %d is below proof height %d", cs.LatestSlot, height.GetRevisionHeight())
	}

	var membershipProof MembershipProof
	if err := cdc.Unmarshal(proof, &membershipProof); err != nil {
		return errorsmod.Wrap(ErrInvalidProof, "failed to unmarshal membership proof")
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	consState, found := GetConsensusState(clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "no consensus state at proof height")
	}

	return verifyStorageNonMembership(cs, consState, &membershipProof, merklePath.KeyPath)
}

// validateHeight checks a proof height: revision zero, non-zero height.
func validateHeight(height exported.Height) error {
	if height == nil {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "height is nil")
	}
	if height.GetRevisionNumber() != 0 {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidHeight, "revision number must be zero, got %d", height.GetRevisionNumber())
	}
	if height.GetRevisionHeight() == 0 {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "height must be non-zero")
	}
	return nil
}
