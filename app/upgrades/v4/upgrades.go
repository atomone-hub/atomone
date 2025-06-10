package v4

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/atomone-hub/atomone/app/keepers"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v4
// This versions contains the upgrade to Cosmos SDK v0.50 and IBC v10
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}
}
