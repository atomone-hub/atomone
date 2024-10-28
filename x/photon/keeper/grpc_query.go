package keeper

import (
	"context"

	"github.com/atomone-hub/atomone/x/photon/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// ConversionRate returns the atone to photon conversion ratio.
func (k Keeper) ConversionRate(goCtx context.Context, req *types.QueryConversionRateRequest) (*types.QueryConversionRateResponse, error) {
	var (
		ctx          = sdk.UnwrapSDKContext(goCtx)
		bondDenom    = k.stakingKeeper.BondDenom(ctx)
		atoneSupply  = k.bankKeeper.GetSupply(ctx, bondDenom).Amount.ToLegacyDec()
		photonSupply = k.bankKeeper.GetSupply(ctx, "uphoton").Amount.ToLegacyDec()
		cr           = k.conversionRate(ctx, atoneSupply, photonSupply)
	)
	return &types.QueryConversionRateResponse{ConversionRate: cr.String()}, nil
}
