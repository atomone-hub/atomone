package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/photon/types"
)

// ConvertToDenom implements the feemarket.types.DenomResolver interface.
// The method is invoked when denom is not the expected fee denom configured
// in the feemarket module (hence `uphoton`). When denom is the bond denom
// (hence `uatone`), the method applies the conversion rate to convert bond
// denom to photon.
// CONTRACT: coin.Denom is the feemarket configured fee denom (`uphoton`).
func (k Keeper) ConvertToDenom(ctx sdk.Context, coin sdk.DecCoin, denom string) (sdk.DecCoin, error) {
	if coin.Denom == denom {
		return coin, nil
	}

	if denom == k.stakingKeeper.BondDenom(ctx) {
		// use the conversion rate to convert bond denom to photon
		bondDenomSupply := k.bankKeeper.GetSupply(ctx, denom).Amount.ToLegacyDec()
		uphotonSupply := k.bankKeeper.GetSupply(ctx, types.Denom).Amount.ToLegacyDec()
		conversionRate := k.conversionRate(ctx, bondDenomSupply, uphotonSupply)

		// convert bond denom to photon
		amount := coin.Amount.Quo(conversionRate)
		return sdk.NewDecCoinFromDec(denom, amount), nil
	}

	return sdk.DecCoin{}, fmt.Errorf("error resolving denom '%s'", denom)
}

// ExtraDenoms implements the feemarket.types.DenomResolver interface.
// The method is expected to returns the other tokens that are allowed to be
// used as a fee token; here only the bond denom.
func (k Keeper) ExtraDenoms(ctx sdk.Context) ([]string, error) {
	return []string{
		k.stakingKeeper.BondDenom(ctx),
	}, nil
}
