package keepers

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	ccvtypes "github.com/cosmos/interchain-security/v5/x/ccv/types"

	atomonegovkeeper "github.com/atomone-hub/atomone/x/gov/keeper"
)

// GovKeeperAdapter adapts AtomOne's gov keeper to match the interface expected by ICS
type GovKeeperAdapter struct {
	keeper *atomonegovkeeper.Keeper
}

// NewGovKeeperAdapter creates a new adapter for AtomOne's gov keeper
func NewGovKeeperAdapter(k *atomonegovkeeper.Keeper) *GovKeeperAdapter {
	return &GovKeeperAdapter{keeper: k}
}

// GetProposal retrieves a proposal and converts it to ICS's Proposal type
func (a *GovKeeperAdapter) GetProposal(ctx sdk.Context, proposalID uint64) (ccvtypes.Proposal, bool) {
	prop, found := a.keeper.GetProposal(ctx, proposalID)
	if !found {
		return ccvtypes.Proposal{}, false
	}

	// Convert AtomOne proposal to ICS proposal - only copy the Messages field
	return ccvtypes.Proposal{
		Messages: prop.Messages,
	}, true
}
