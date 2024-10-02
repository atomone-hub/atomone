package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/photon/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Burn(goCtx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.MintDisabled {
		return nil, types.ErrMintDisabled
	}

	// Ensure burned amount denom is bond denom (atone)
	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if msg.Amount.Denom != bondDenom {
		return nil, types.ErrMintInvalidDenom
	}
	// Send amount to photon module
	to, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, err
	}
	coins := sdk.NewCoins(msg.Amount)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, to, types.ModuleName, coins); err != nil {
		return nil, err
	}
	// Burn amount
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, err
	}
	// Compute photons to mint
	// TODO

	return &types.MsgBurnResponse{}, nil
}
