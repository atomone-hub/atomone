package types

import (
	"cosmossdk.io/errors"
)

// x/gov module sentinel errors
var (
	ErrInactiveProposal      = errors.Register(ModuleName, 30, "inactive proposal")       //nolint:staticcheck
	ErrAlreadyActiveProposal = errors.Register(ModuleName, 40, "proposal already active") //nolint:staticcheck
	// Errors 5 & 6 are legacy errors related to v1beta1.Proposal.
	ErrInvalidProposalContent       = errors.Register(ModuleName, 50, "invalid proposal content")                                  //nolint:staticcheck
	ErrInvalidProposalType          = errors.Register(ModuleName, 60, "invalid proposal type")                                     //nolint:staticcheck
	ErrInvalidVote                  = errors.Register(ModuleName, 70, "invalid vote option")                                       //nolint:staticcheck
	ErrInvalidGenesis               = errors.Register(ModuleName, 80, "invalid genesis state")                                     //nolint:staticcheck
	ErrNoProposalHandlerExists      = errors.Register(ModuleName, 90, "no handler exists for proposal type")                       //nolint:staticcheck
	ErrUnroutableProposalMsg        = errors.Register(ModuleName, 100, "proposal message not recognized by router")                //nolint:staticcheck
	ErrNoProposalMsgs               = errors.Register(ModuleName, 110, "no messages proposed")                                     //nolint:staticcheck
	ErrInvalidProposalMsg           = errors.Register(ModuleName, 120, "invalid proposal message")                                 //nolint:staticcheck
	ErrInvalidSigner                = errors.Register(ModuleName, 130, "expected gov account as only signer for proposal message") //nolint:staticcheck
	ErrInvalidSignalMsg             = errors.Register(ModuleName, 140, "signal message is invalid")                                //nolint:staticcheck
	ErrMetadataTooLong              = errors.Register(ModuleName, 150, "metadata too long")                                        //nolint:staticcheck
	ErrMinDepositTooSmall           = errors.Register(ModuleName, 160, "minimum deposit is too small")                             //nolint:staticcheck
	ErrInvalidConstitutionAmendment = errors.Register(ModuleName, 170, "invalid constitution amendment")                           //nolint:staticcheck
	ErrUnknownProposal              = errors.Register(ModuleName, 180, "unknown proposal")
)
