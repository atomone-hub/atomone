package keeper

import (
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/crisis/types"
)

// new crisis genesis
func (k *Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) {
	if err := k.SetConstantFee(ctx, data.ConstantFee); err != nil {
		panic(err)
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	constantFee := k.GetConstantFee(ctx)
	return types.NewGenesisState(constantFee)
}
