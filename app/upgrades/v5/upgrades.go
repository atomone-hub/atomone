package v5

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/atomone-hub/atomone/app/keepers"
)

// CreateUpgradeHandler returns the upgrade handler for AtomOne v5, which runs
// the pending module migrations, including the x/mint migration to time-based
// minting.
func CreateUpgradeHandler(
	mm *module.Manager,
	_ codec.Codec,
	configurator module.Configurator,
	_ *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
