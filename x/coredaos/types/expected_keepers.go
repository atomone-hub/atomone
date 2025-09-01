package types

import (
	"context"
	time "time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// GovKeeper defines the expected interface needed to interact with the
// governance module.
type GovKeeper interface {
	// GetProposalID gets the highest proposal ID
	GetProposalID(ctx sdk.Context) (proposalID uint64, err error)
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

	// DeleteVotes deletes all votes from a proposal with given proposalID
	DeleteVotes(ctx sdk.Context, proposalID uint64)
}

// StakingKeeper defines the expected interface needed to interact with the
// staking module.
type StakingKeeper interface {
	// GetDelegatorBonded returns the total amount a delegator has bonded.
	GetDelegatorBonded(ctx context.Context, delegator sdk.AccAddress) (math.Int, error)
	// GetDelegatorUnbonding returns the total amount a delegator has unbonding.
	GetDelegatorUnbonding(ctx context.Context, delegator sdk.AccAddress) (math.Int, error)
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// BankKeeper defines the expected account keeper used for simulations (noalias)
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}
