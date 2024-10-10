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

const photonMaxSupply = 1_000_000_000

func (k msgServer) Burn(goCtx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.MintDisabled {
		return nil, types.ErrMintDisabled
	}

	// Ensure burned amount denom is bond denom (uatone)
	// TODO ensure it is uatone
	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if msg.Amount.Denom != bondDenom {
		return nil, types.ErrBurnInvalidDenom
	}
	// Compute photons to mint
	var (
		atomToBurn            = msg.Amount
		atoneSupply           = k.bankKeeper.GetSupply(ctx, bondDenom)
		photonSupply          = k.bankKeeper.GetSupply(ctx, "uphoton")
		remainMintablePhotons = sdk.NewInt(photonMaxSupply).Sub(photonSupply.Amount)
		photonToMint          = atomToBurn.Amount.Mul(
			remainMintablePhotons.Quo(atoneSupply.Amount),
		)
	)

	if photonToMint.IsZero() {
		return nil, types.ErrNoMintablePhotons
	}
	// TODO check if photonToMint + remainMintablePhotons > photonMaxSupply ?
	to, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, err
	}

	// Send atone to photon module for burn
	coinsToBurn := sdk.NewCoins(atomToBurn)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, to, types.ModuleName, coinsToBurn); err != nil {
		return nil, err
	}
	// Burn atone
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coinsToBurn); err != nil {
		return nil, err
	}

	// Mint photons
	coinsToMint := sdk.NewCoins(sdk.NewCoin("uphoton", photonToMint))
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coinsToMint); err != nil {
		return nil, err
	}
	// Send minted photon to account
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to, coinsToMint); err != nil {
		return nil, err
	}

	return &types.MsgBurnResponse{Minted: coinsToMint[0]}, nil
}
