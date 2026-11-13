package spectre

import (
	"context"

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
	ctx context.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	consState, found := GetConsensusState(ctx, clientStore, cdc, height)
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
		{"gloas", fp.Gloas.Version},
	}
	for _, v := range versions {
		if len(v.version) != 4 {
			return errorsmod.Wrapf(ErrInvalidClientState, "%s fork version must be 4 bytes", v.name)
		}
	}

	// Fork epochs must be monotonically non-decreasing. Equal epochs are valid:
	// a devnet that starts every fork at genesis schedules them all at epoch 0,
	// and computeForkVersion then selects the latest of them, as the consensus
	// specs do. An unscheduled fork is epoch 2^64 - 1.
	epochs := []uint64{
		fp.Altair.Epoch, fp.Bellatrix.Epoch, fp.Capella.Epoch,
		fp.Deneb.Epoch, fp.Electra.Epoch, fp.Fulu.Epoch, fp.Gloas.Epoch,
	}
	for i := 1; i < len(epochs); i++ {
		if epochs[i] < epochs[i-1] {
			return errorsmod.Wrap(ErrInvalidClientState, "fork epochs must be non-decreasing")
		}
	}
	return nil
}

// initialize stores the initial client and consensus states.
func (cs *ClientState) initialize(
	ctx sdk.Context,
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

	if cs.Frozen {
		return errorsmod.Wrap(ErrInvalidClientState, "a new client state must not be frozen")
	}
	// A client that does not know its next sync committee cannot cross into the
	// next sync committee period, and the committee it learns from a later
	// update is only trusted to follow the period of its own anchor.
	if consensusState.NextSyncCommittee == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "next sync committee must be set")
	}

	if cs.LatestSlot != consensusState.Slot {
		return errorsmod.Wrapf(ErrClientAndConsensusMismatch,
			"client latest slot %d does not match consensus slot %d", cs.LatestSlot, consensusState.Slot)
	}

	setClientState(ctx, clientStore, cdc, cs)
	setConsensusState(ctx, clientStore, cdc, consensusState, cs.GetLatestHeight())
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
		// A header names the consensus state it is verified against, but
		// updateState applies the result on top of the latest one. Verifying
		// against an older state would let the keys of a past sync committee,
		// which carry no stake any more, forge a root and write it at a slot that
		// is already stored. Misbehaviour evidence writes nothing and has its
		// own rule, see verifyMisbehaviour.
		if msg.TrustedSlot != cs.LatestSlot {
			return errorsmod.Wrapf(ErrTrustedSlotNotLatest, "trusted slot %d, latest slot %d", msg.TrustedSlot, cs.LatestSlot)
		}
		consState, found := GetConsensusState(ctx, clientStore, cdc, clienttypes.NewHeight(0, msg.TrustedSlot))
		if !found {
			return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "trusted slot %d", msg.TrustedSlot)
		}
		// Charge for the committee BLS work before any of it runs: the
		// size-based gas of the message does not cover it, and verification
		// may fail only at the pairing check, deep inside the expensive part.
		// The charge follows the non-signers, whose keys the verifier decodes.
		ctx.GasMeter().ConsumeGas(cs.GasCostHeaderVerification(msg.ConsensusUpdate.SyncAggregate.SyncCommitteeBits), "spectre: header verification")
		return cs.verifyHeader(consState, uint64(ctx.BlockTime().Unix()), msg)
	case *Misbehaviour:
		consState, found := GetConsensusState(ctx, clientStore, cdc, clienttypes.NewHeight(0, msg.TrustedSlot))
		if !found {
			return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "trusted slot %d", msg.TrustedSlot)
		}
		ctx.GasMeter().ConsumeGas(
			cs.GasCostMisbehaviourVerification(msg.Update_1.SyncAggregate.SyncCommitteeBits, msg.Update_2.SyncAggregate.SyncCommitteeBits),
			"spectre: misbehaviour verification")
		return cs.verifyMisbehaviour(consState, uint64(ctx.BlockTime().Unix()), msg)
	default:
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected type %T or %T, got %T",
			&Header{}, &Misbehaviour{}, clientMsg)
	}
}

// checkForMisbehaviour reports whether the client message is a valid
// misbehaviour report. ibc-go's 02-client keeper only calls it from
// UpdateClient, immediately after VerifyClientMessage accepted the same
// message, so a Misbehaviour reaching this point has already passed the full
// verification, including both BLS aggregate checks; re-running them would
// double the cost of the most expensive client message for no new
// information. A Header is never a misbehaviour report.
func (cs *ClientState) checkForMisbehaviour(clientMsg exported.ClientMessage) bool {
	if cs.Frozen {
		return false
	}
	_, ok := clientMsg.(*Misbehaviour)
	return ok
}

// updateState applies a verified header, storing the new consensus state and
// updating the client state when the finalized slot advanced. Updates apply
// against the consensus state at the client's latest slot; historical
// updates are not supported. Consensus states below the pruning window are
// pruned here, inside the relayer's metered transaction, like the tendermint
// client; a client that stops updating stops growing, so nothing needs a
// per-block sweeper.
//
// A Misbehaviour is not applied here: the 02-client keeper only reaches this
// with one after CheckForMisbehaviour reported it, which freezes the client
// through UpdateStateOnMisbehaviour instead. But the RedundantRelayDecorator
// calls UpdateState during CheckTx for every top-level MsgUpdateClient with
// misbehaviour checks omitted, so a non-header must return no heights rather
// than panic, like the tendermint client.
//
// The panics below are invariant guards: updateConsensusState fails only on
// trusted/latest inconsistencies that verifyHeader rules out for verified
// headers (a nil next sync committee exists only in the client's initial
// sync period, where the signature period check pins updates to that
// period). They must never fire; if one does, the verification invariants
// were broken by a code change.
func (cs *ClientState) updateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	clientMsg exported.ClientMessage,
) []exported.Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		// clientMsg is a Misbehaviour, no update necessary
		return []exported.Height{}
	}

	// The header was verified against its trusted slot and is applied on top of
	// the latest consensus state. verifyClientMessage only accepts the two being
	// the same; this keeps the apply step from relying on that having run.
	if header.TrustedSlot != cs.LatestSlot {
		panic(errorsmod.Wrapf(ErrTrustedSlotNotLatest, "trusted slot %d, latest slot %d", header.TrustedSlot, cs.LatestSlot))
	}

	// The update applies against the consensus state at the client's latest
	// slot; historical updates are not supported.
	consState, found := GetConsensusState(ctx, clientStore, cdc, cs.GetLatestHeight())
	if !found {
		panic(errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "latest slot %d", cs.LatestSlot))
	}

	newConsState, newClientState, err := cs.updateConsensusState(consState, header)
	if err != nil {
		panic(err)
	}

	height := clienttypes.NewHeight(0, newConsState.Slot)
	setConsensusState(ctx, clientStore, cdc, newConsState, height)
	if newClientState != nil {
		setClientState(ctx, clientStore, cdc, newClientState)
	}

	// Prune below the window, like the tendermint client: skipped in CheckTx
	// and ReCheckTx for performance, but not in simulation, so gas estimation
	// sees the pruning cost. The store operations are gas-metered through the
	// provider, so the scan and deletions are paid for by the update tx.
	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() || ctx.ExecMode() == sdk.ExecModeSimulate {
		if _, err := PruneConsensusStates(ctx, clientStore, cdc, cs.PruneKeepSlots(), pruneBatchLimit); err != nil {
			ctx.Logger().Error("failed to prune spectre consensus states", "error", err)
		}
	}
	return []exported.Height{height}
}

// updateStateOnMisbehaviour freezes the client.
func (cs *ClientState) updateStateOnMisbehaviour(
	ctx context.Context,
	cdc codec.BinaryCodec,
	clientStore storetypes.KVStore,
	_ exported.ClientMessage,
) {
	if cs.Frozen {
		return
	}
	frozen := cs.Clone()
	frozen.Frozen = true
	setClientState(ctx, clientStore, cdc, frozen)
}

// verifyMembership verifies a storage membership proof of value at path at
// the given height against the IBC router contract on Ethereum.
func (cs *ClientState) verifyMembership(
	ctx sdk.Context,
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
	if err := cs.validateProofHeightAboveWindow(height); err != nil {
		return err
	}

	var membershipProof MembershipProof
	if err := cdc.Unmarshal(proof, &membershipProof); err != nil {
		return errorsmod.Wrap(ErrInvalidProof, "failed to unmarshal membership proof")
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	consState, found := GetConsensusState(ctx, clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "no consensus state at proof height")
	}

	return verifyStorageMembership(cs, consState, &membershipProof, merklePath.KeyPath, value)
}

// verifyNonMembership verifies a storage non-membership proof at path at
// the given height.
func (cs *ClientState) verifyNonMembership(
	ctx sdk.Context,
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
	if err := cs.validateProofHeightAboveWindow(height); err != nil {
		return err
	}

	var membershipProof MembershipProof
	if err := cdc.Unmarshal(proof, &membershipProof); err != nil {
		return errorsmod.Wrap(ErrInvalidProof, "failed to unmarshal membership proof")
	}

	merklePath, ok := path.(commitmenttypesv2.MerklePath)
	if !ok {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", commitmenttypesv2.MerklePath{}, path)
	}

	consState, found := GetConsensusState(ctx, clientStore, cdc, height)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "no consensus state at proof height")
	}

	return verifyStorageNonMembership(cs, consState, &membershipProof, merklePath.KeyPath)
}

// validateProofHeightAboveWindow rejects proof heights below the pruning
// window. Consensus states below the window are pruned, so acceptance must
// not depend on whether pruning has run: the rule is the window itself.
func (cs *ClientState) validateProofHeightAboveWindow(height exported.Height) error {
	keepSlots := cs.PruneKeepSlots()
	if cs.LatestSlot > keepSlots && height.GetRevisionHeight() < cs.LatestSlot-keepSlots {
		return errorsmod.Wrapf(ErrProofHeightBelowWindow,
			"proof height %d is below the pruning window [%d, %d]",
			height.GetRevisionHeight(), cs.LatestSlot-keepSlots, cs.LatestSlot)
	}
	return nil
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
