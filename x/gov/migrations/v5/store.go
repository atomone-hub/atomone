package v5

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

var ParamsKey = []byte{0x30}

// Addition of the dynamic-deposit parameters.
// Addition of the burnDepositNoThreshold parameter.
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)
	paramsBz := store.Get(ParamsKey)

	var params govv1.Params
	cdc.MustUnmarshal(paramsBz, &params)

	defaultParams := govv1.DefaultParams()
	params.MinDepositThrottler = defaultParams.MinDepositThrottler
	params.MinInitialDepositThrottler = defaultParams.MinInitialDepositThrottler
	params.BurnDepositNoThreshold = defaultParams.BurnDepositNoThreshold

	bz, err := cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(ParamsKey, bz)
	return nil
}
