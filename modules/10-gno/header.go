package gno

import (
	"bytes"
	"errors"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"
)

var _ exported.ClientMessage = (*Header)(nil)

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

	// Convert and validate signed header
	gnoSignedHeader, err := ConvertToGnoSignedHeader(h.SignedHeader)
	if err != nil {
		return errorsmod.Wrap(err, "failed to convert signed header")
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

	// Convert and validate validator set
	gnoValset, err := ConvertToGnoValidatorSet(h.ValidatorSet)
	if err != nil {
		return err
	}

	if !bytes.Equal(h.SignedHeader.Header.ValidatorsHash, gnoValset.Hash()) {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator set does not match hash")
	}
	return nil
}
