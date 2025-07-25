package types

import (
	time "time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// GovKeeper defines the expected interface needed to retrieve account balances.
type GovKeeper interface {
	// GetProposal gets a proposal from store by ProposalID.
	GetProposal(ctx sdk.Context, id uint64) (govtypesv1.Proposal, bool)
	// SetProposal sets a proposal in the gov store.
	SetProposal(ctx sdk.Context, proposal govtypesv1.Proposal)
	// ProposalKinds returns a v1.ProposalKinds useful to determine which kind
	// of messages are included in a proposal.
	ProposalKinds(proposal govtypesv1.Proposal) govtypesv1.ProposalKinds
	// DeleteAndBurnDeposits deletes and burns all the deposits on a
	// specific proposal.
	DeleteAndBurnDeposits(ctx sdk.Context, proposalID uint64)
	// RefundAndDeleteDeposits refunds and deletes all the deposits on a
	// specific proposal.
	RefundAndDeleteDeposits(ctx sdk.Context, proposalID uint64)
	// InsertActiveProposalQueue inserts a proposalID into the active proposal
	// queue at endTime
	InsertActiveProposalQueue(ctx sdk.Context, proposalID uint64, endTime time.Time)
	// RemoveFromActiveProposalQueue removes a proposalID from the Active
	// Proposal Queue
	RemoveFromActiveProposalQueue(ctx sdk.Context, proposalID uint64, endTime time.Time)
	// DecrementActiveProposalsNumber decrements the number of active proposals
	// by one
	DecrementActiveProposalsNumber(ctx sdk.Context)
	// UpdateMinInitialDeposit updates the min initial deposit required for
	// proposal submission
	UpdateMinInitialDeposit(ctx sdk.Context, checkElapsedTime bool)
	// UpdateMinDeposit updates the minimum deposit required for a proposal
	UpdateMinDeposit(ctx sdk.Context, checkElapsedTime bool)
}
