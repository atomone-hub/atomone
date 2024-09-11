package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/gov module sentinel errors
var (
	ErrUnknownProposal       = sdkerrors.Register(ModuleName, 20, "unknown proposal")
	ErrInactiveProposal      = sdkerrors.Register(ModuleName, 30, "inactive proposal")
	ErrAlreadyActiveProposal = sdkerrors.Register(ModuleName, 40, "proposal already active")
	// Errors 5 & 6 are legacy errors related to v1beta1.Proposal.
	ErrInvalidProposalContent  = sdkerrors.Register(ModuleName, 50, "invalid proposal content")
	ErrInvalidProposalType     = sdkerrors.Register(ModuleName, 60, "invalid proposal type")
	ErrInvalidVote             = sdkerrors.Register(ModuleName, 70, "invalid vote option")
	ErrInvalidGenesis          = sdkerrors.Register(ModuleName, 80, "invalid genesis state")
	ErrNoProposalHandlerExists = sdkerrors.Register(ModuleName, 90, "no handler exists for proposal type")
	ErrUnroutableProposalMsg   = sdkerrors.Register(ModuleName, 100, "proposal message not recognized by router")
	ErrNoProposalMsgs          = sdkerrors.Register(ModuleName, 110, "no messages proposed")
	ErrInvalidProposalMsg      = sdkerrors.Register(ModuleName, 120, "invalid proposal message")
	ErrInvalidSigner           = sdkerrors.Register(ModuleName, 130, "expected gov account as only signer for proposal message")
	ErrInvalidSignalMsg        = sdkerrors.Register(ModuleName, 140, "signal message is invalid")
	ErrMetadataTooLong         = sdkerrors.Register(ModuleName, 150, "metadata too long")
	ErrMinDepositTooSmall      = sdkerrors.Register(ModuleName, 160, "minimum deposit is too small")
)
