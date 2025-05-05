package v3

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/atomone-hub/atomone/app/keepers"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v3
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Starting module migrations...")
		// RunMigrations will detect the add of the feemarket module, will initiate
		// its genesis and will fill the versionMap with its consensus version.
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		ctx.Logger().Info("Upgrade complete")
		return vm, nil
	}
}
