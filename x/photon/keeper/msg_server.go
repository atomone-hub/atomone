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

// Burn implements the MsgServer.Burn method.
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
		atoneToBurn  = msg.Amount
		photonToMint = atoneToBurn.Amount.ToLegacyDec().Mul(k.conversionRate(ctx))
	)

	if photonToMint.IsZero() {
		return nil, types.ErrNoMintablePhotons
	}
	// TODO check if photonToMint + remainMintablePhotons > photonMaxSupply ?
	// TODO we probably needs to deal with round precision

	// Send atone to photon module for burn
	to, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, err
	}
	coinsToBurn := sdk.NewCoins(atoneToBurn)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, to, types.ModuleName, coinsToBurn); err != nil {
		return nil, err
	}
	// Burn atone
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coinsToBurn); err != nil {
		return nil, err
	}

	// Mint photons
	coinsToMint := sdk.NewCoins(sdk.NewCoin("uphoton", photonToMint.RoundInt()))
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coinsToMint); err != nil {
		return nil, err
	}
	// Send minted photon to account
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to, coinsToMint); err != nil {
		return nil, err
	}

	return &types.MsgBurnResponse{Minted: coinsToMint[0]}, nil
}

const photonMaxSupply = 1_000_000_000

// conversionRate returns the conversion rate for converting atone to photon.
func (k Keeper) conversionRate(ctx sdk.Context) sdk.Dec {
	var (
		bondDenom             = k.stakingKeeper.BondDenom(ctx)
		atoneSupply           = k.bankKeeper.GetSupply(ctx, bondDenom).Amount.ToLegacyDec()
		photonSupply          = k.bankKeeper.GetSupply(ctx, "uphoton").Amount.ToLegacyDec()
		remainMintablePhotons = sdk.NewDec(photonMaxSupply).Sub(photonSupply)
	)
	return remainMintablePhotons.Quo(atoneSupply)
}
