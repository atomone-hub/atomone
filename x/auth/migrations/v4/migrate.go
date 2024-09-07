package v4

import (
	"github.com/atomone-hub/atomone/codec"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/auth/exported"
	"github.com/atomone-hub/atomone/x/auth/types"
)

var ParamsKey = []byte{0x00}

// Migrate migrates the x/auth module state from the consensus version 3 to
// version 4. Specifically, it takes the parameters that are currently stored
// and managed by the x/params modules and stores them directly into the x/auth
// module state.
func Migrate(ctx sdk.Context, store sdk.KVStore, legacySubspace exported.Subspace, cdc codec.BinaryCodec) error {
	var currParams types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	if err := currParams.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&currParams)
	store.Set(ParamsKey, bz)

	return nil
}
