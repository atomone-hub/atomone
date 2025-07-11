package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/photon/types"
)

// ConvertToDenom returns "coin.Amount denom" for all coins that are not the denom.
func (k Keeper) ConvertToDenom(ctx sdk.Context, coin sdk.DecCoin, denom string) (sdk.DecCoin, error) {
	if coin.Denom == denom {
		return coin, nil
	}

	if denom == k.stakingKeeper.BondDenom(ctx) {
		// use the conversion rate to convert bond denom to photon
		bondDenomSupply := k.bankKeeper.GetSupply(ctx, denom).Amount.ToLegacyDec()
		uphotonSupply := k.bankKeeper.GetSupply(ctx, types.Denom).Amount.ToLegacyDec()
		conversionRate := k.PhotonConversionRate(ctx, bondDenomSupply, uphotonSupply)

		// convert bond denom to photon
		amount := coin.Amount.Quo(conversionRate)
		return sdk.NewDecCoinFromDec(denom, amount), nil
	}

	return sdk.DecCoin{}, fmt.Errorf("error resolving denom")
}

func (k Keeper) ExtraDenoms(ctx sdk.Context) ([]string, error) {
	return []string{
		k.stakingKeeper.BondDenom(ctx),
	}, nil
}
