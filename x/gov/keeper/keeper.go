package keeper

import (
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	"github.com/atomone-hub/atomone/x/gov/types"
)

// Keeper defines the governance module Keeper
type Keeper struct {
	*govkeeper.Keeper
}

// NewKeeper returns a governance keeper. It wraps the orginal Atom One SDK module for backward compatibility.
func NewKeeper(k *govkeeper.Keeper) *Keeper {
	return &Keeper{
		Keeper: k,
	}
}

// Logger returns a module-specific logger.
func (keeper Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
