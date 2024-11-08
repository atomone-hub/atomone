package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/photon/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// ConversionRate returns the staking denom to photon conversion ratio.
func (k Keeper) ConversionRate(goCtx context.Context, req *types.QueryConversionRateRequest) (*types.QueryConversionRateResponse, error) {
	var (
		ctx                = sdk.UnwrapSDKContext(goCtx)
		bondDenom          = k.stakingKeeper.BondDenom(ctx)
		stakingDenomSupply = k.bankKeeper.GetSupply(ctx, bondDenom).Amount.ToLegacyDec()
		uphotonSupply      = k.bankKeeper.GetSupply(ctx, types.Denom).Amount.ToLegacyDec()
		cr                 = k.conversionRate(ctx, stakingDenomSupply, uphotonSupply)
	)
	return &types.QueryConversionRateResponse{ConversionRate: cr.String()}, nil
}
