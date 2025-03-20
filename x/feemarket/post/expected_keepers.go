package post

import (
	"github.com/atomone-hub/atomone/x/feemarket/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type FeeMarketKeeper interface {
	GetState(ctx sdk.Context) (types.State, error)
	GetParams(ctx sdk.Context) (types.Params, error)
	SetState(ctx sdk.Context, state types.State) error
	GetEnabledHeight(ctx sdk.Context) (int64, error)
}
