package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/gov module sentinel errors
var (
	ErrUnknownProposal       = sdkerrors.Register(ModuleName, 20, "unknown proposal")        //nolint:staticcheck
	ErrInactiveProposal      = sdkerrors.Register(ModuleName, 30, "inactive proposal")       //nolint:staticcheck
	ErrAlreadyActiveProposal = sdkerrors.Register(ModuleName, 40, "proposal already active") //nolint:staticcheck
	// Errors 5 & 6 are legacy errors related to v1beta1.Proposal.
	ErrInvalidProposalContent  = sdkerrors.Register(ModuleName, 50, "invalid proposal content")                                  //nolint:staticcheck
	ErrInvalidProposalType     = sdkerrors.Register(ModuleName, 60, "invalid proposal type")                                     //nolint:staticcheck
	ErrInvalidVote             = sdkerrors.Register(ModuleName, 70, "invalid vote option")                                       //nolint:staticcheck
	ErrInvalidGenesis          = sdkerrors.Register(ModuleName, 80, "invalid genesis state")                                     //nolint:staticcheck
	ErrNoProposalHandlerExists = sdkerrors.Register(ModuleName, 90, "no handler exists for proposal type")                       //nolint:staticcheck
	ErrUnroutableProposalMsg   = sdkerrors.Register(ModuleName, 100, "proposal message not recognized by router")                //nolint:staticcheck
	ErrNoProposalMsgs          = sdkerrors.Register(ModuleName, 110, "no messages proposed")                                     //nolint:staticcheck
	ErrInvalidProposalMsg      = sdkerrors.Register(ModuleName, 120, "invalid proposal message")                                 //nolint:staticcheck
	ErrInvalidSigner           = sdkerrors.Register(ModuleName, 130, "expected gov account as only signer for proposal message") //nolint:staticcheck
	ErrInvalidSignalMsg        = sdkerrors.Register(ModuleName, 140, "signal message is invalid")                                //nolint:staticcheck
	ErrMetadataTooLong         = sdkerrors.Register(ModuleName, 150, "metadata too long")                                        //nolint:staticcheck
	ErrMinDepositTooSmall      = sdkerrors.Register(ModuleName, 160, "minimum deposit is too small")                             //nolint:staticcheck
)
