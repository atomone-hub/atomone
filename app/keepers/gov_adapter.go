package keepers

import (
	ccvtypes "github.com/allinbits/interchain-security/x/ccv/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	atomonegovkeeper "github.com/atomone-hub/atomone/x/gov/keeper"
)

// GovKeeperAdapter adapts AtomOne's custom governance keeper to the interface
// expected by the ICS provider keeper. This is necessary because AtomOne uses
// a forked governance module with different proposal structures.
type GovKeeperAdapter struct {
	keeper *atomonegovkeeper.Keeper
}

// NewGovKeeperAdapter creates a new governance keeper adapter.
func NewGovKeeperAdapter(k *atomonegovkeeper.Keeper) *GovKeeperAdapter {
	return &GovKeeperAdapter{keeper: k}
}

// GetProposal retrieves a proposal from AtomOne's governance keeper and converts
// it to the format expected by ICS. Currently only copies the Messages field.
func (a *GovKeeperAdapter) GetProposal(ctx sdk.Context, proposalID uint64) (ccvtypes.Proposal, bool) {
	prop, found := a.keeper.GetProposal(ctx, proposalID)
	if !found {
		return ccvtypes.Proposal{}, false
	}
	return ccvtypes.Proposal{
		Messages: prop.Messages,
	}, true
}
