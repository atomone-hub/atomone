package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

var _ GovKeeper = (*Keeper)(nil)

// GovKeeper defines the expected interface needed to interact with the
// governance module.
type GovKeeper interface {
	// GetProposalID gets the highest proposal ID
	GetProposalID(ctx sdk.Context) (proposalID uint64, err error)
	// GetProposal gets a proposal from store by ProposalID.
	GetProposal(ctx sdk.Context, id uint64) (v1.Proposal, bool)
	// SetProposal sets a proposal in the gov store.
	SetProposal(ctx sdk.Context, proposal v1.Proposal)
	// ProposalKinds returns a v1.ProposalKinds useful to determine which kind
	// of messages are included in a proposal.
	ProposalKinds(proposal v1.Proposal) v1.ProposalKinds
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
	// UpdateMinInitialDeposit updates the min initial deposit required for
	// proposal submission
	UpdateMinInitialDeposit(ctx sdk.Context, checkElapsedTime bool)
	// UpdateMinDeposit updates the minimum deposit required for a proposal
	UpdateMinDeposit(ctx sdk.Context, checkElapsedTime bool)
	// DeleteVotes deletes all votes from a proposal with given proposalID
	DeleteVotes(ctx sdk.Context, proposalID uint64)
}
