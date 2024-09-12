package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/gov module sentinel errors
var (
	ErrUnknownProposal       = sdkerrors.Register(ModuleName, 2, "unknown proposal")        //nolint:staticcheck // SA1019
	ErrInactiveProposal      = sdkerrors.Register(ModuleName, 3, "inactive proposal")       //nolint:staticcheck // SA1019
	ErrAlreadyActiveProposal = sdkerrors.Register(ModuleName, 4, "proposal already active") //nolint:staticcheck // SA1019
	// Errors 5 & 6 are legacy errors related to v1beta1.Proposal.
	ErrInvalidProposalContent  = sdkerrors.Register(ModuleName, 5, "invalid proposal content")                                  //nolint:staticcheck // SA1019
	ErrInvalidProposalType     = sdkerrors.Register(ModuleName, 6, "invalid proposal type")                                     //nolint:staticcheck // SA1019
	ErrInvalidVote             = sdkerrors.Register(ModuleName, 7, "invalid vote option")                                       //nolint:staticcheck // SA1019
	ErrInvalidGenesis          = sdkerrors.Register(ModuleName, 8, "invalid genesis state")                                     //nolint:staticcheck // SA1019
	ErrNoProposalHandlerExists = sdkerrors.Register(ModuleName, 9, "no handler exists for proposal type")                       //nolint:staticcheck // SA1019
	ErrUnroutableProposalMsg   = sdkerrors.Register(ModuleName, 10, "proposal message not recognized by router")                //nolint:staticcheck // SA1019
	ErrNoProposalMsgs          = sdkerrors.Register(ModuleName, 11, "no messages proposed")                                     //nolint:staticcheck // SA1019
	ErrInvalidProposalMsg      = sdkerrors.Register(ModuleName, 12, "invalid proposal message")                                 //nolint:staticcheck // SA1019
	ErrInvalidSigner           = sdkerrors.Register(ModuleName, 13, "expected gov account as only signer for proposal message") //nolint:staticcheck // SA1019
	ErrInvalidSignalMsg        = sdkerrors.Register(ModuleName, 14, "signal message is invalid")                                //nolint:staticcheck // SA1019
	ErrMetadataTooLong         = sdkerrors.Register(ModuleName, 15, "metadata too long")                                        //nolint:staticcheck // SA1019
	ErrMinDepositTooSmall      = sdkerrors.Register(ModuleName, 16, "minimum deposit is too small")                             //nolint:staticcheck // SA1019
)
