package keeper

import (
	sdk "github.com/atomone-hub/atomone/types"
	v2 "github.com/atomone-hub/atomone/x/group/migrations/v2"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.Migrate(
		ctx,
		m.keeper.key,
		m.keeper.accKeeper,
		m.keeper.groupPolicySeq,
		m.keeper.groupPolicyTable,
	)
}
