package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"

	"github.com/atomone-hub/atomone/x/coredaos/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	params, err := k.Params.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		panic(err)
	}
	return params
}
