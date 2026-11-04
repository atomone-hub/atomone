package v5

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/atomone-hub/atomone/app/keepers"
	"github.com/atomone-hub/atomone/app/upgrades"
	"github.com/atomone-hub/atomone/modules/spectre"
)

// CreateUpgradeHandler returns the upgrade handler for AtomOne v5.
// This version registers the spectre Ethereum light client.
func CreateUpgradeHandler(
	mm *module.Manager,
	cdc codec.Codec,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		if err := upgrades.AppendAllowedClient(ctx, keepers.IBCKeeper, spectre.ModuleName); err != nil {
			return vm, err
		}

		return vm, nil
	}
}
