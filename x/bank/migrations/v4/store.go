package v4

import (
	"github.com/atomone-hub/atomone/codec"
	storetypes "github.com/atomone-hub/atomone/store/types"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/bank/exported"
	"github.com/atomone-hub/atomone/x/bank/types"
)

const ModuleName = "bank"

var ParamsKey = []byte{0x05}

// MigrateStore migrates the x/bank module state from the consensus version 3 to
// version 4. Specifically, it takes the parameters that are currently stored
// and managed by the x/params module and stores them directly into the x/bank
// module state.
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, legacySubspace exported.Subspace, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(storeKey)
	var currParams types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	// SendEnabled is migrated to the x/bank module store, so delete from the params
	currParams = types.NewParams(currParams.DefaultSendEnabled)

	if err := currParams.Validate(); err != nil {
		return err
	}

	bz, err := cdc.Marshal(&currParams)
	if err != nil {
		return err
	}

	store.Set(ParamsKey, bz)

	return nil
}