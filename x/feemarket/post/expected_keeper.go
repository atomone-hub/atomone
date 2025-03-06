package post

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	feemarkettypes "github.com/atomone-hub/atomone/x/feemarket/types"
)

// FeeMarketKeeper defines the expected feemarket keeper.
//
//go:generate mockery --name FeeMarketKeeper --filename mock_feemarket_keeper.go
type FeemarketKeeper interface {
	GetState(ctx sdk.Context) (feemarkettypes.State, error)
	GetParams(ctx sdk.Context) (feemarkettypes.Params, error)
	SetState(ctx sdk.Context, state feemarkettypes.State) error
	GetEnabledHeight(ctx sdk.Context) (int64, error)
}
