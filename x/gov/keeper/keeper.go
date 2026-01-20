package keeper

import (
	"time"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// Keeper defines the governance module Keeper
type Keeper struct {
	*govkeeper.Keeper
}

// NewKeeper returns a governance keeper. It wraps the original Atom One SDK module for backward compatibility.
func NewKeeper(k *govkeeper.Keeper) *Keeper {
	return &Keeper{
		Keeper: k,
	}
}

// DeleteAndBurnDeposits implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).DeleteAndBurnDeposits of Keeper.Keeper.
func (keeper *Keeper) DeleteAndBurnDeposits(ctx sdk.Context, proposalID uint64) {
	_ = keeper.Keeper.DeleteAndBurnDeposits(ctx, proposalID)
}

// DeleteVotes implements GovKeeper.
func (keeper *Keeper) DeleteVotes(ctx sdk.Context, proposalID uint64) {
	// Delete all votes for the proposal using collections API
	// Votes are stored as collections.Map[collections.Pair[uint64, sdk.AccAddress], v1.Vote]
	// We need to clear all votes with the given proposalID prefix
	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposalID)
	_ = keeper.Keeper.Votes.Clear(ctx, rng)
}

// GetProposal implements GovKeeper.
func (keeper *Keeper) GetProposal(ctx sdk.Context, id uint64) (v1.Proposal, bool) {
	sdkProposal, err := keeper.Keeper.Proposals.Get(ctx, id)
	if err != nil {
		return v1.Proposal{}, false
	}
	atomoneProposal := v1.ConvertSDKProposalToAtomOne(&sdkProposal)
	if atomoneProposal == nil {
		return v1.Proposal{}, false
	}
	return *atomoneProposal, true
}

// GetProposalID implements GovKeeper.
func (keeper *Keeper) GetProposalID(ctx sdk.Context) (proposalID uint64, err error) {
	return keeper.Keeper.ProposalID.Peek(ctx)
}

// InsertActiveProposalQueue implements GovKeeper.
func (keeper *Keeper) InsertActiveProposalQueue(ctx sdk.Context, proposalID uint64, endTime time.Time) {
	// ActiveProposalsQueue is collections.Map[collections.Pair[time.Time, uint64], uint64]
	key := collections.Join(endTime, proposalID)
	_ = keeper.Keeper.ActiveProposalsQueue.Set(ctx, key, proposalID)
}

// ProposalKinds implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).ProposalKinds of Keeper.Keeper.
func (keeper *Keeper) ProposalKinds(proposal v1.Proposal) v1.ProposalKinds {
	sdkProposal := v1.ConvertAtomOneProposalToSDK(&proposal)
	return v1.ProposalKinds(keeper.Keeper.ProposalKinds(*sdkProposal))
}

// RefundAndDeleteDeposits implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).RefundAndDeleteDeposits of Keeper.Keeper.
func (keeper *Keeper) RefundAndDeleteDeposits(ctx sdk.Context, proposalID uint64) {
	_ = keeper.Keeper.RefundAndDeleteDeposits(ctx, proposalID)
}

// RemoveFromActiveProposalQueue implements GovKeeper.
func (keeper *Keeper) RemoveFromActiveProposalQueue(ctx sdk.Context, proposalID uint64, endTime time.Time) {
	// ActiveProposalsQueue is collections.Map[collections.Pair[time.Time, uint64], uint64]
	key := collections.Join(endTime, proposalID)
	_ = keeper.Keeper.ActiveProposalsQueue.Remove(ctx, key)
}

// SetProposal implements GovKeeper.
func (keeper *Keeper) SetProposal(ctx sdk.Context, proposal v1.Proposal) {
	sdkProposal := v1.ConvertAtomOneProposalToSDK(&proposal)
	_ = keeper.Keeper.SetProposal(ctx, *sdkProposal)
}

// UpdateMinDeposit implements GovKeeper.
func (keeper *Keeper) UpdateMinDeposit(ctx sdk.Context, checkElapsedTime bool) {
	keeper.Keeper.UpdateMinDeposit(ctx, checkElapsedTime)
}

// UpdateMinInitialDeposit implements GovKeeper.
func (keeper *Keeper) UpdateMinInitialDeposit(ctx sdk.Context, checkElapsedTime bool) {
	keeper.Keeper.UpdateMinInitialDeposit(ctx, checkElapsedTime)
}
