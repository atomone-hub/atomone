package keeper

import (
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

// GetConstantFee get's the constant fee from the store
func (k *Keeper) GetConstantFee(ctx sdk.Context) (constantFee sdk.Coin) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConstantFeeKey)
	if bz == nil {
		return constantFee
	}
	k.cdc.MustUnmarshal(bz, &constantFee)
	return constantFee
}

// GetConstantFee set's the constant fee in the store
func (k *Keeper) SetConstantFee(ctx sdk.Context, constantFee sdk.Coin) error {
	if !constantFee.IsValid() || constantFee.IsNegative() {
		return errors.Wrapf(errors.ErrInvalidCoins, "negative or invalid constant fee: %s", constantFee) //nolint: staticcheck
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&constantFee)
	if err != nil {
		return err
	}

	store.Set(types.ConstantFeeKey, bz)
	return nil
}
