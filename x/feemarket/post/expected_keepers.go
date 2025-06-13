package post

import (
	"context"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/feemarket/types"
)

type FeeMarketKeeper interface {
	GetState(ctx sdk.Context) (types.State, error)
	GetParams(ctx sdk.Context) (types.Params, error)
	SetState(ctx sdk.Context, state types.State) error
	GetEnabledHeight(ctx sdk.Context) (int64, error)
}

type ConsensusParamsKeeper interface {
	Get(context.Context) (tmproto.ConsensusParams, error)
}
