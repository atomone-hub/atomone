package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/exported"
	v5 "github.com/atomone-hub/atomone/x/gov/migrations/v5"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper         *Keeper
	legacySubspace exported.ParamSubspace
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper, legacySubspace exported.ParamSubspace) Migrator {
	return Migrator{
		keeper:         keeper,
		legacySubspace: legacySubspace,
	}
}

// Migrate4to5 migrates the store from version 4 to 5.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	return v5.MigrateStore(ctx, m.keeper.storeKey, m.keeper.cdc)
}
