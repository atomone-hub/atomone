package types

import (
	"cosmossdk.io/errors"
)

// x/gov module sentinel errors
var (
	ErrInactiveProposal      = errors.Register(ModuleName, 30, "inactive proposal")
	ErrAlreadyActiveProposal = errors.Register(ModuleName, 40, "proposal already active")
	// Errors 5 & 6 are legacy errors related to v1beta1.Proposal.
	ErrInvalidProposalContent         = errors.Register(ModuleName, 50, "invalid proposal content")
	ErrInvalidProposalType            = errors.Register(ModuleName, 60, "invalid proposal type")
	ErrInvalidVote                    = errors.Register(ModuleName, 70, "invalid vote option")
	ErrInvalidGenesis                 = errors.Register(ModuleName, 80, "invalid genesis state")
	ErrNoProposalHandlerExists        = errors.Register(ModuleName, 90, "no handler exists for proposal type")
	ErrUnroutableProposalMsg          = errors.Register(ModuleName, 100, "proposal message not recognized by router")
	ErrNoProposalMsgs                 = errors.Register(ModuleName, 110, "no messages proposed")
	ErrInvalidProposalMsg             = errors.Register(ModuleName, 120, "invalid proposal message")
	ErrInvalidSigner                  = errors.Register(ModuleName, 130, "expected gov account as only signer for proposal message")
	ErrInvalidSignalMsg               = errors.Register(ModuleName, 140, "signal message is invalid")
	ErrMetadataTooLong                = errors.Register(ModuleName, 150, "metadata too long")
	ErrMinDepositTooSmall             = errors.Register(ModuleName, 160, "minimum deposit is too small")
	ErrInvalidConstitutionAmendment   = errors.Register(ModuleName, 170, "invalid constitution amendment")
	ErrUnknownProposal                = errors.Register(ModuleName, 180, "unknown proposal")
	ErrGovernorExists                 = errors.Register(ModuleName, 300, "governor already exists")
	ErrGovernorNotFound               = errors.Register(ModuleName, 310, "governor not found")
	ErrInvalidGovernorStatus          = errors.Register(ModuleName, 320, "invalid governor status")
	ErrGovernanceDelegationExists     = errors.Register(ModuleName, 330, "governance delegation already exists")
	ErrGovernanceDelegationNotFound   = errors.Register(ModuleName, 340, "governance delegation not found")
	ErrInvalidGovernanceDescription   = errors.Register(ModuleName, 350, "invalid governance description")
	ErrDelegatorIsGovernor            = errors.Register(ModuleName, 360, "cannot explicitly manage governance delegations, delegator is an active governor")
	ErrGovernorStatusEqual            = errors.Register(ModuleName, 370, "cannot change governor status to the same status")
	ErrGovernorStatusChangePeriod     = errors.Register(ModuleName, 380, "governor status change period not elapsed")
	ErrInsufficientGovernorDelegation = errors.Register(ModuleName, 390, "insufficient governor self-delegation")
)
