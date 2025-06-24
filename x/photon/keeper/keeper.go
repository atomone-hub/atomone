package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/photon/types"
)

type Keeper struct {
	cdc       codec.BinaryCodec
	storeKey  storetypes.StoreKey
	authority string

	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
	stakingKeeper types.StakingKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority string,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		authority:     authority,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
		stakingKeeper: stakingKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// conversionRate returns the conversion rate for converting bond denom to
// photon.
func (k Keeper) conversionRate(_ sdk.Context, bondDenomSupply, uphotonSupply math.LegacyDec) math.LegacyDec {
	remainMintableUphotons := math.LegacyNewDec(types.MaxSupply).Sub(uphotonSupply)
	if remainMintableUphotons.IsNegative() {
		// If for any reason the max supply is exceeded, avoid returning a negative number
		return math.LegacyZeroDec()
	}
	return remainMintableUphotons.Quo(bondDenomSupply)
}
