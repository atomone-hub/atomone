package gno

import (
	"bytes"
	"errors"
	"fmt"

	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VerifyClientMessage checks if the clientMessage is of type Header or Misbehaviour and verifies the message
func (cs *ClientState) VerifyClientMessage(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore,
	clientMsg exported.ClientMessage,
) error {
	switch msg := clientMsg.(type) {
	case *Header:
		return cs.verifyHeader(ctx, clientStore, cdc, msg)
	case *Misbehaviour:
		return cs.verifyMisbehaviour(ctx, clientStore, cdc, msg)
	default:
		return clienttypes.ErrInvalidClientType
	}
}

// verifyHeader returns an error if:
// - the client or header provided are not parseable to gno types
// - the header is invalid
// - header height is less than or equal to the trusted header height
// - header revision is not equal to trusted header revision
// - header valset commit verification fails
// - header timestamp is past the trusting period in relation to the consensus state
// - header timestamp is less than or equal to the consensus state timestamp
func (cs *ClientState) verifyHeader(
	ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec,
	header *Header,
) error {
	currentTimestamp := ctx.BlockTime()
	// Retrieve trusted consensus states for each Header in misbehaviour
	consState, found := GetConsensusState(clientStore, cdc, header.TrustedHeight)
	if !found {
		return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header at TrustedHeight: %s", header.TrustedHeight)
	}
	if err := checkTrustedHeader(header, consState); err != nil {
		return err
	}

	// UpdateClient only accepts updates with a header at the same revision
	// as the trusted consensus state
	if header.GetHeight().GetRevisionNumber() != header.TrustedHeight.RevisionNumber {
		return errorsmod.Wrapf(
			ErrInvalidHeaderHeight,
			"header height revision %d does not match trusted header revision %d",
			header.GetHeight().GetRevisionNumber(), header.TrustedHeight.RevisionNumber,
		)
	}

	gnoTrustedValidators := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(header.TrustedValidators.Validators)),
		Proposer:   nil,
	}
	for i, val := range header.TrustedValidators.Validators {
		key := val.PubKey
		if (key.GetEd25519()) == nil {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		gnoTrustedValidators.Validators[i] = &bfttypes.Validator{
			Address:          crypto.MustAddressFromString(val.Address),
			PubKey:           ed25519.PubKeyEd25519(key.GetEd25519()),
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}
	gnoTrustedValidators.TotalVotingPower() // ensure TotalVotingPower is set
	if gnoTrustedValidators.IsNilOrEmpty() {
		return errorsmod.Wrap(errors.New("not gno validators"), "trusted validator set in not gno validator set type")
	}
	var dataHash []byte
	var lastResultsHash []byte
	if len(header.SignedHeader.Header.DataHash) == 0 {
		dataHash = nil
	} else {
		dataHash = header.SignedHeader.Header.DataHash
	}
	if len(header.SignedHeader.Header.LastResultsHash) == 0 {
		lastResultsHash = nil
	} else {
		lastResultsHash = header.SignedHeader.Header.LastResultsHash
	}
	gnoHeader := bfttypes.Header{
		Version:            header.SignedHeader.Header.Version,
		ChainID:            header.SignedHeader.Header.ChainId,
		Height:             header.SignedHeader.Header.Height,
		Time:               header.SignedHeader.Header.Time,
		NumTxs:             header.SignedHeader.Header.NumTxs,
		TotalTxs:           header.SignedHeader.Header.TotalTxs,
		LastBlockID:        bfttypes.BlockID{Hash: header.SignedHeader.Header.LastBlockId.Hash, PartsHeader: bfttypes.PartSetHeader{Total: int(header.SignedHeader.Header.LastBlockId.PartsHeader.Total), Hash: header.SignedHeader.Header.LastBlockId.PartsHeader.Hash}},
		LastCommitHash:     header.SignedHeader.Header.LastCommitHash,
		DataHash:           dataHash,
		ValidatorsHash:     header.SignedHeader.Header.ValidatorsHash,
		NextValidatorsHash: header.SignedHeader.Header.NextValidatorsHash,
		ConsensusHash:      header.SignedHeader.Header.ConsensusHash,
		AppHash:            header.SignedHeader.Header.AppHash,
		LastResultsHash:    lastResultsHash,
		ProposerAddress:    crypto.MustAddressFromString(header.SignedHeader.Header.ProposerAddress),
	}
	gnoCommit := bfttypes.Commit{
		BlockID:    bfttypes.BlockID{Hash: header.SignedHeader.Commit.BlockId.Hash, PartsHeader: bfttypes.PartSetHeader{Total: int(header.SignedHeader.Commit.BlockId.PartsHeader.Total), Hash: header.SignedHeader.Commit.BlockId.PartsHeader.Hash}},
		Precommits: make([]*bfttypes.CommitSig, len(header.SignedHeader.Commit.Precommits)),
	}
	for i, sig := range header.SignedHeader.Commit.Precommits {
		if sig == nil {
			continue
		}
		gnoCommit.Precommits[i] = &bfttypes.CommitSig{
			ValidatorIndex:   int(sig.ValidatorIndex),
			Signature:        sig.Signature,
			BlockID:          bfttypes.BlockID{Hash: sig.BlockId.Hash, PartsHeader: bfttypes.PartSetHeader{Total: int(sig.BlockId.PartsHeader.Total), Hash: sig.BlockId.PartsHeader.Hash}},
			Type:             bfttypes.SignedMsgType(sig.Type),
			Height:           sig.Height,
			Round:            int(sig.Round),
			Timestamp:        sig.Timestamp,
			ValidatorAddress: crypto.MustAddressFromString(sig.ValidatorAddress),
		}
	}
	gnoSignedHeader := bfttypes.SignedHeader{
		Header: &gnoHeader,
		Commit: &gnoCommit,
	}
	err := gnoSignedHeader.ValidateBasic(header.SignedHeader.Header.ChainId) // ensure signed header is valid
	if err != nil {
		return errorsmod.Wrap(err, "signed header in not gno signed header type")
	}

	gnoValidatorSet := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(header.ValidatorSet.Validators)),
		Proposer:   nil,
	}
	for i, val := range header.ValidatorSet.Validators {
		key := val.PubKey
		if (key.GetEd25519()) == nil {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		gnoValidatorSet.Validators[i] = &bfttypes.Validator{
			Address:          crypto.MustAddressFromString(val.Address),
			PubKey:           ed25519.PubKeyEd25519(key.GetEd25519()),
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}
	gnoValidatorSet.TotalVotingPower() // ensure TotalVotingPower is set
	if gnoValidatorSet.IsNilOrEmpty() {
		return errorsmod.Wrap(errors.New("not gno validators"), "trusted validator set in not gno validator set type")
	}

	// assert header height is newer than consensus state
	if header.GetHeight().LTE(header.TrustedHeight) {
		return errorsmod.Wrapf(
			clienttypes.ErrInvalidHeader,
			"header height ≤ consensus state height (%s ≤ %s)", header.GetHeight(), header.TrustedHeight,
		)
	}

	// Construct a trusted header using the fields in consensus state
	// Only Height, Time, and NextValidatorsHash are necessary for verification
	// NOTE: updates must be within the same revision
	trustedHeader := bfttypes.Header{
		ChainID:            cs.GetChainID(),
		Height:             int64(header.TrustedHeight.RevisionHeight),
		Time:               consState.Timestamp,
		NextValidatorsHash: consState.NextValidatorsHash,
	}
	signedHeader := bfttypes.SignedHeader{
		Header: &trustedHeader,
	}

	// Verify next header with the passed-in trustedVals
	// - asserts trusting period not passed
	// - assert header timestamp is not past the trusting period
	// - assert header timestamp is past latest stored consensus state timestamp
	// - assert that a TrustLevel proportion of TrustedValidators signed new Commit

	// TODO: replace with gno light client verification
	err = Verify(
		&signedHeader,
		&gnoTrustedValidators, &gnoSignedHeader, &gnoValidatorSet,
		cs.TrustingPeriod, currentTimestamp, cs.MaxClockDrift, cs.TrustLevel.ToTendermint(),
	)
	if err != nil {
		return errorsmod.Wrap(err, "failed to verify header")
	}

	return nil
}

// UpdateState may be used to either create a consensus state for:
// - a future height greater than the latest client state height
// - a past height that was skipped during bisection
// If we are updating to a past height, a consensus state is created for that height to be persisted in client store
// If we are updating to a future height, the consensus state is created and the client state is updated to reflect
// the new latest height
// A list containing the updated consensus height is returned.
// UpdateState must only be used to update within a single revision, thus header revision number and trusted height's revision
// number must be the same. To update to a new revision, use a separate upgrade path
// UpdateState will prune the oldest consensus state if it is expired.
// If the provided clientMsg is not of type of Header then the handler will noop and empty slice is returned.
func (cs ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	header, ok := clientMsg.(*Header)
	if !ok {
		// clientMsg is invalid Misbehaviour, no update necessary
		return []exported.Height{}
	}

	// performance: do not prune in checkTx
	// simulation must prune for accurate gas estimation
	if (!ctx.IsCheckTx() && !ctx.IsReCheckTx()) || ctx.ExecMode() == sdk.ExecModeSimulate {
		cs.pruneOldestConsensusState(ctx, cdc, clientStore)
	}

	// check for duplicate update
	if _, found := GetConsensusState(clientStore, cdc, header.GetHeight()); found {
		// perform no-op
		return []exported.Height{header.GetHeight()}
	}

	height, ok := header.GetHeight().(clienttypes.Height)
	if !ok {
		panic(fmt.Errorf("cannot convert %T to %T", header.GetHeight(), &clienttypes.Height{}))
	}
	if height.GT(cs.LatestHeight) {
		cs.LatestHeight = height
	}

	consensusState := &ConsensusState{
		Timestamp:          header.GetTime(),
		Root:               commitmenttypes.NewMerkleRoot(header.SignedHeader.Header.AppHash),
		NextValidatorsHash: header.SignedHeader.Header.NextValidatorsHash,
	}

	// set client state, consensus state and associated metadata
	setClientState(clientStore, cdc, &cs)
	setConsensusState(clientStore, cdc, consensusState, header.GetHeight())
	setConsensusMetadata(ctx, clientStore, header.GetHeight())

	return []exported.Height{height}
}

// pruneOldestConsensusState will retrieve the earliest consensus state for this clientID and check if it is expired. If it is,
// that consensus state will be pruned from store along with all associated metadata. This will prevent the client store from
// becoming bloated with expired consensus states that can no longer be used for updates and packet verification.
func (cs ClientState) pruneOldestConsensusState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore) {
	// Check the earliest consensus state to see if it is expired, if so then set the prune height
	// so that we can delete consensus state and all associated metadata.
	var (
		pruneHeight exported.Height
	)

	pruneCb := func(height exported.Height) bool {
		consState, found := GetConsensusState(clientStore, cdc, height)
		// this error should never occur
		if !found {
			panic(errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "failed to retrieve consensus state at height: %s", height))
		}

		if cs.IsExpired(consState.Timestamp, ctx.BlockTime()) {
			pruneHeight = height
		}

		return true
	}

	IterateConsensusStateAscending(clientStore, pruneCb)

	// if pruneHeight is set, delete consensus state and metadata
	if pruneHeight != nil {
		deleteConsensusState(clientStore, pruneHeight)
		deleteConsensusMetadata(clientStore, pruneHeight)
	}
}

// UpdateStateOnMisbehaviour updates state upon misbehaviour, freezing the ClientState. This method should only be called when misbehaviour is detected
// as it does not perform any misbehaviour checks.
func (cs ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, _ exported.ClientMessage) {
	cs.FrozenHeight = FrozenHeight

	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, &cs))
}

// checkTrustedHeader checks that consensus state matches trusted fields of Header
func checkTrustedHeader(header *Header, consState *ConsensusState) error {
	gnoTrustedValset := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(header.TrustedValidators.Validators)),
		Proposer:   nil,
	}
	for i, val := range header.TrustedValidators.Validators {
		key := val.PubKey
		if (key.GetEd25519()) == nil {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		gnoTrustedValset.Validators[i] = &bfttypes.Validator{
			Address:          crypto.MustAddressFromString(val.Address),
			PubKey:           ed25519.PubKeyEd25519(key.GetEd25519()),
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}
	gnoTrustedValset.TotalVotingPower() // ensure TotalVotingPower is set
	if gnoTrustedValset.IsNilOrEmpty() {
		return errorsmod.Wrap(errors.New("empty trusted validator set"), "trusted validator set is not gno validator set type")
	}

	// assert that trustedVals is NextValidators of last trusted header
	// to do this, we check that trustedVals.Hash() == consState.NextValidatorsHash
	tvalHash := gnoTrustedValset.Hash()
	if !bytes.Equal(consState.NextValidatorsHash, tvalHash) {
		return errorsmod.Wrapf(
			ErrInvalidValidatorSet,
			"trusted validators %s, does not hash to latest trusted validators. Expected: %X, got: %X",
			header.TrustedValidators, consState.NextValidatorsHash, tvalHash,
		)
	}
	return nil
}
