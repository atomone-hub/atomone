package post

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/dynamicfee/types"
)

type DynamicfeeKeeper interface {
	GetState(ctx sdk.Context) (types.State, error)
	GetParams(ctx sdk.Context) (types.Params, error)
	GetMaxBlockGas(ctx sdk.Context, params types.Params) uint64
	SetState(ctx sdk.Context, state types.State) error
	GetEnabledHeight(ctx sdk.Context) (int64, error)
}
