package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// Keeper defines the governance module Keeper
type Keeper struct {
	*govkeeper.Keeper
}

// NewKeeper returns a governance keeper. It wraps the orginal Atom One SDK module for backward compatibility.
func NewKeeper(k *govkeeper.Keeper) *Keeper {
	return &Keeper{
		Keeper: k,
	}
}

// DecrementActiveProposalsNumber implements GovKeeper.
func (keeper *Keeper) DecrementActiveProposalsNumber(ctx sdk.Context) {
	panic("unimplemented")
}

// DeleteAndBurnDeposits implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).DeleteAndBurnDeposits of Keeper.Keeper.
func (keeper *Keeper) DeleteAndBurnDeposits(ctx sdk.Context, proposalID uint64) {
	panic("unimplemented")
}

// DeleteVotes implements GovKeeper.
func (keeper *Keeper) DeleteVotes(ctx sdk.Context, proposalID uint64) {
	panic("unimplemented")
}

// GetProposal implements GovKeeper.
func (keeper *Keeper) GetProposal(ctx sdk.Context, id uint64) (v1.Proposal, bool) {
	panic("unimplemented")
}

// GetProposalID implements GovKeeper.
func (keeper *Keeper) GetProposalID(ctx sdk.Context) (proposalID uint64, err error) {
	panic("unimplemented")
}

// InsertActiveProposalQueue implements GovKeeper.
func (keeper *Keeper) InsertActiveProposalQueue(ctx sdk.Context, proposalID uint64, endTime time.Time) {
	panic("unimplemented")
}

// ProposalKinds implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).ProposalKinds of Keeper.Keeper.
func (keeper *Keeper) ProposalKinds(proposal v1.Proposal) v1.ProposalKinds {
	panic("unimplemented")
}

// RefundAndDeleteDeposits implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).RefundAndDeleteDeposits of Keeper.Keeper.
func (keeper *Keeper) RefundAndDeleteDeposits(ctx sdk.Context, proposalID uint64) {
	panic("unimplemented")
}

// RemoveFromActiveProposalQueue implements GovKeeper.
func (keeper *Keeper) RemoveFromActiveProposalQueue(ctx sdk.Context, proposalID uint64, endTime time.Time) {
	panic("unimplemented")
}

// SetProposal implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).SetProposal of Keeper.Keeper.
func (keeper *Keeper) SetProposal(ctx sdk.Context, proposal v1.Proposal) {
	panic("unimplemented")
}

// UpdateMinDeposit implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).UpdateMinDeposit of Keeper.Keeper.
func (keeper *Keeper) UpdateMinDeposit(ctx sdk.Context, checkElapsedTime bool) {
	panic("unimplemented")
}

// UpdateMinInitialDeposit implements GovKeeper.
// Subtle: this method shadows the method (*Keeper).UpdateMinInitialDeposit of Keeper.Keeper.
func (keeper *Keeper) UpdateMinInitialDeposit(ctx sdk.Context, checkElapsedTime bool) {
	panic("unimplemented")
}
