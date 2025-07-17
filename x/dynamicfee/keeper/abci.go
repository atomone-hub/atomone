package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlock returns an endblocker for the x/dynamicfee module. The endblocker
// is responsible for updating the state of the dynamic fee pricing based on
// the AIMD learning rate adjustment algorithm.
func (k *Keeper) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	if err := k.UpdateDynamicfee(ctx); err != nil {
		panic(err)
	}

	return []abci.ValidatorUpdate{}
}
