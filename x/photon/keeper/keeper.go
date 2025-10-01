package keeper

import (
	"context"
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

// PhotonConversionRate returns the conversion rate for converting bond denom to
// photon.
// NOTE: bondDenomSupply cannot be zero when the chain is producing blocks (thus it can never be zero).
// This is because the only way for validators to be able to participate in block production is to have
// staked bond denom, which therefore is locked and cannot be burned. Although this condition is logically
// impossible, we still add a panic here to be defensive.
func (k Keeper) PhotonConversionRate(_ context.Context, bondDenomSupply, uphotonSupply math.LegacyDec) math.LegacyDec {
	remainMintableUphotons := math.LegacyNewDec(types.MaxSupply).Sub(uphotonSupply)
	if remainMintableUphotons.IsNegative() {
		// If for any reason the max supply is exceeded, avoid returning a negative number
		return math.LegacyZeroDec()
	}
	if bondDenomSupply.IsZero() {
		panic("bond denom supply cannot be zero")
	}
	return remainMintableUphotons.Quo(bondDenomSupply)
}
