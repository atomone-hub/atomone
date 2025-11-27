package gno

import (
	"bytes"
	"errors"
	"reflect"
	"time"

	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CheckForMisbehaviour detects duplicate height misbehaviour and BFT time violation misbehaviour
// in a submitted Header message and verifies the correctness of a submitted Misbehaviour ClientMessage
func (ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore, msg exported.ClientMessage) bool {
	switch msg := msg.(type) {
	case *Header:
		header := msg
		consState := header.ConsensusState()

		// Check if the Client store already has a consensus state for the header's height
		// If the consensus state exists, and it matches the header then we return early
		// since header has already been submitted in a previous UpdateClient.
		if existingConsState, found := GetConsensusState(clientStore, cdc, header.GetHeight()); found {
			// This header has already been submitted and the necessary state is already stored
			// in client store, thus we can return early without further validation.
			if reflect.DeepEqual(existingConsState, header.ConsensusState()) {
				return false
			}

			// A consensus state already exists for this height, but it does not match the provided header.
			// The assumption is that Header has already been validated. Thus we can return true as misbehaviour is present
			return true
		}

		// Check that consensus state timestamps are monotonic
		prevCons, prevOk := GetPreviousConsensusState(clientStore, cdc, header.GetHeight())
		nextCons, nextOk := GetNextConsensusState(clientStore, cdc, header.GetHeight())
		// if previous consensus state exists, check consensus state time is greater than previous consensus state time
		// if previous consensus state is not before current consensus state return true
		if prevOk && !prevCons.Timestamp.Before(consState.Timestamp) {
			return true
		}
		// if next consensus state exists, check consensus state time is less than next consensus state time
		// if next consensus state is not after current consensus state return true
		if nextOk && !nextCons.Timestamp.After(consState.Timestamp) {
			return true
		}
	case *Misbehaviour:
		// if heights are equal check that this is valid misbehaviour of a fork
		// otherwise if heights are unequal check that this is valid misbehavior of BFT time violation
		if msg.Header1.GetHeight().EQ(msg.Header2.GetHeight()) {

			blockID1 := bfttypes.BlockID{
				Hash:        msg.Header1.SignedHeader.Header.LastBlockId.Hash,
				PartsHeader: bfttypes.PartSetHeader{Total: int(msg.Header1.SignedHeader.Header.LastBlockId.PartsHeader.Total), Hash: msg.Header1.SignedHeader.Header.LastBlockId.PartsHeader.Hash},
			}
			if blockID1.ValidateBasic() != nil {
				return false
			}
			blockID2 := bfttypes.BlockID{
				Hash:        msg.Header2.SignedHeader.Header.LastBlockId.Hash,
				PartsHeader: bfttypes.PartSetHeader{Total: int(msg.Header2.SignedHeader.Header.LastBlockId.PartsHeader.Total), Hash: msg.Header2.SignedHeader.Header.LastBlockId.PartsHeader.Hash},
			}
			if blockID2.ValidateBasic() != nil {
				return false
			}

			// Ensure that Commit Hashes are different
			if !bytes.Equal(blockID1.Hash, blockID2.Hash) {
				return true
			}

		} else if !msg.Header1.SignedHeader.Header.Time.After(msg.Header2.SignedHeader.Header.Time) {
			// Header1 is at greater height than Header2, therefore Header1 time must be less than or equal to
			// Header2 time in order to be valid misbehaviour (violation of monotonic time).
			return true
		}
	}

	return false
}

// verifyMisbehaviour determines whether or not two conflicting
// headers at the same height would have convinced the light client.
//
// NOTE: consensusState1 is the trusted consensus state that corresponds to the TrustedHeight
// of misbehaviour.Header1
// Similarly, consensusState2 is the trusted consensus state that corresponds
// to misbehaviour.Header2
// Misbehaviour sets frozen height to {0, 1} since it is only used as a boolean value (zero or non-zero).
func (cs *ClientState) verifyMisbehaviour(ctx sdk.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, misbehaviour *Misbehaviour) error {
	// Regardless of the type of misbehaviour, ensure that both headers are valid and would have been accepted by light-client

	// Retrieve trusted consensus states for each Header in misbehaviour
	tmConsensusState1, found := GetConsensusState(clientStore, cdc, misbehaviour.Header1.TrustedHeight)
	if !found {
		return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header1 at TrustedHeight: %s", misbehaviour.Header1.TrustedHeight)
	}

	tmConsensusState2, found := GetConsensusState(clientStore, cdc, misbehaviour.Header2.TrustedHeight)
	if !found {
		return errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "could not get trusted consensus state from clientStore for Header2 at TrustedHeight: %s", misbehaviour.Header2.TrustedHeight)
	}

	// Check the validity of the two conflicting headers against their respective
	// trusted consensus states
	// NOTE: header height and commitment root assertions are checked in
	// misbehaviour.ValidateBasic by the client keeper and msg.ValidateBasic
	// by the base application.
	if err := checkMisbehaviourHeader(
		cs, tmConsensusState1, misbehaviour.Header1, ctx.BlockTime(),
	); err != nil {
		return errorsmod.Wrap(err, "verifying Header1 in Misbehaviour failed")
	}
	if err := checkMisbehaviourHeader(
		cs, tmConsensusState2, misbehaviour.Header2, ctx.BlockTime(),
	); err != nil {
		return errorsmod.Wrap(err, "verifying Header2 in Misbehaviour failed")
	}

	return nil
}

// checkMisbehaviourHeader checks that a Header in Misbehaviour is valid misbehaviour given
// a trusted ConsensusState
func checkMisbehaviourHeader(
	clientState *ClientState, consState *ConsensusState, header *Header, currentTimestamp time.Time,
) error {
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
	err := gnoCommit.ValidateBasic()
	if err != nil {
		return errorsmod.Wrap(err, "commit is not gno commit type")
	}

	// check the trusted fields for the header against ConsensusState
	if err := checkTrustedHeader(header, consState); err != nil {
		return err
	}

	// assert that the age of the trusted consensus state is not older than the trusting period
	if currentTimestamp.Sub(consState.Timestamp) >= clientState.TrustingPeriod {
		return errorsmod.Wrapf(
			ErrTrustingPeriodExpired,
			"current timestamp minus the latest consensus state timestamp is greater than or equal to the trusting period (%d >= %d)",
			currentTimestamp.Sub(consState.Timestamp), clientState.TrustingPeriod,
		)
	}

	chainID := clientState.GetChainID()
	// If chainID is in revision format, then set revision number of chainID with the revision number
	// of the misbehaviour header
	// NOTE: misbehaviour verification is not supported for chains which upgrade to a new chainID without
	// strictly following the chainID revision format
	if clienttypes.IsRevisionFormat(chainID) {
		chainID, _ = clienttypes.SetRevisionNumber(chainID, header.GetHeight().GetRevisionNumber())
	}

	// - ValidatorSet must have TrustLevel similarity with trusted ValidatorSet
	// - TODO: re-check
	err = VerifyLightCommit(&gnoTrustedValset, chainID, gnoCommit.BlockID, header.SignedHeader.Header.Height, &gnoCommit, LCDefaultTrustLevel)
	if err != nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidMisbehaviour, "validator set in header has too much change from trusted validator set: %v", err)
	}
	return nil
}
