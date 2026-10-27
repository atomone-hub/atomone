package v41

import (
	"context"

	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/atomone-hub/atomone/app/keepers"
	"github.com/atomone-hub/atomone/app/upgrades"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v4.1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

// CreateUpgradeHandler is a stub handler to reconcicle the appVersion discrepency between AtomOne testnet and AtomOne mainnet.
// On mainnet this upgrade handler does nothing. Check https://github.com/atomone-hub/atomone/commit/f45a45e4475aa210cfb3687b11c09b93a611d50c for testnet reference.
// From this point both testnet and mainnet can have the same binary.
func CreateUpgradeHandler(
	_ *module.Manager,
	_ codec.Codec,
	_ module.Configurator,
	_ *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return vm, nil
	}
}
