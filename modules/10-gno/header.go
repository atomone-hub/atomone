package gno

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	errorsmod "cosmossdk.io/errors"
)

var _ exported.ClientMessage = (*Header)(nil)

func hexDec(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// ConsensusState returns the updated consensus state associated with the header
func (h Header) ConsensusState() *ConsensusState {
	return &ConsensusState{
		Timestamp:          h.GetTime(),
		Root:               commitmenttypes.NewMerkleRoot(h.SignedHeader.Header.AppHash),
		NextValidatorsHash: h.SignedHeader.Header.NextValidatorsHash,
	}
}

// ClientType defines that the Header is a Gno consensus algorithm
func (Header) ClientType() string {
	return Gno
}

// GetHeight returns the current height. It returns 0 if the gno
// header is nil.
// NOTE: the header.Header is checked to be non nil in ValidateBasic.
func (h Header) GetHeight() exported.Height {
	revision := clienttypes.ParseChainID(h.SignedHeader.Header.ChainId)
	return clienttypes.NewHeight(revision, uint64(h.SignedHeader.Header.Height))
}

// GetTime returns the current block timestamp. It returns a zero time if
// the gno header is nil.
// NOTE: the header.Header is checked to be non nil in ValidateBasic.
func (h Header) GetTime() time.Time {
	return h.SignedHeader.Header.Time
}

// ValidateBasic calls the SignedHeader ValidateBasic function and checks
// that validatorsets are not nil.
// NOTE: TrustedHeight and TrustedValidators may be empty when creating client
// with MsgCreateClient
func (h Header) ValidateBasic() error {
	if h.SignedHeader == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "gno signed header cannot be nil")
	}
	// SignedHeader ValidateBasic() checks that Header and Commit are not nil
	if h.SignedHeader.Header == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "gno header cannot be nil")
	}
	if h.SignedHeader.Commit == nil {
		return errorsmod.Wrap(errors.New("missing commit"), "gno commit cannot be nil")
	}
	// GnoHeader ValidateBasic()
	gnoHeader := bfttypes.Header{
		Version:            h.SignedHeader.Header.Version,
		ChainID:            h.SignedHeader.Header.ChainId,
		Height:             h.SignedHeader.Header.Height,
		Time:               h.SignedHeader.Header.Time,
		NumTxs:             h.SignedHeader.Header.NumTxs,
		TotalTxs:           h.SignedHeader.Header.TotalTxs,
		LastBlockID:        bfttypes.BlockID{Hash: h.SignedHeader.Header.LastBlockId.Hash, PartsHeader: bfttypes.PartSetHeader{Total: int(h.SignedHeader.Header.LastBlockId.PartsHeader.Total), Hash: h.SignedHeader.Header.LastBlockId.PartsHeader.Hash}},
		LastCommitHash:     h.SignedHeader.Header.LastCommitHash,
		DataHash:           h.SignedHeader.Header.DataHash,
		ValidatorsHash:     h.SignedHeader.Header.ValidatorsHash,
		NextValidatorsHash: h.SignedHeader.Header.NextValidatorsHash,
		ConsensusHash:      h.SignedHeader.Header.ConsensusHash,
		AppHash:            h.SignedHeader.Header.AppHash,
		LastResultsHash:    h.SignedHeader.Header.LastResultsHash,
		ProposerAddress:    crypto.MustAddressFromString(h.SignedHeader.Header.ProposerAddress),
	}
	gnoCommit := bfttypes.Commit{
		BlockID:    bfttypes.BlockID{Hash: h.SignedHeader.Commit.BlockId.Hash, PartsHeader: bfttypes.PartSetHeader{Total: int(h.SignedHeader.Commit.BlockId.PartsHeader.Total), Hash: h.SignedHeader.Commit.BlockId.PartsHeader.Hash}},
		Precommits: make([]*bfttypes.CommitSig, len(h.SignedHeader.Commit.Precommits)),
	}
	for i, sig := range h.SignedHeader.Commit.Precommits {
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
	// NOTE: SignedHeader ValidateBasic checks
	if err := gnoSignedHeader.ValidateBasic(h.SignedHeader.Header.ChainId); err != nil {
		return errorsmod.Wrap(err, "header failed basic validation")
	}

	// TrustedHeight is less than Header for updates and misbehaviour
	if h.TrustedHeight.GTE(h.GetHeight()) {
		return errorsmod.Wrapf(ErrInvalidHeaderHeight, "TrustedHeight %d must be less than header height %d",
			h.TrustedHeight, h.GetHeight())
	}

	if h.ValidatorSet == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator set is nil")
	}
	gnoValset := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(h.ValidatorSet.Validators)),
		Proposer:   nil,
	}
	for i, val := range h.ValidatorSet.Validators {
		key := val.PubKey
		if (key.GetEd25519()) == nil {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		gnoValset.Validators[i] = &bfttypes.Validator{
			Address:          crypto.MustAddressFromString(val.Address),
			PubKey:           ed25519.PubKeyEd25519(key.GetEd25519()),
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}
	gnoValset.TotalVotingPower() // ensure TotalVotingPower is set

	if !bytes.Equal(h.SignedHeader.Header.ValidatorsHash, gnoValset.Hash()) {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator set does not match hash")
	}
	return nil
}
