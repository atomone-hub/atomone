package v3

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/atomone-hub/atomone/app/keepers"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v3
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		sdkCtx.Logger().Info("Starting module migrations...")
		// RunMigrations will detect the add of the feemarket module, will initiate
		// its genesis and will fill the versionMap with its consensus version.
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		sdkCtx.Logger().Info("Upgrade complete")
		return vm, nil
	}
}
