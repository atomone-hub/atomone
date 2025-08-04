package v4

import (
	store "cosmossdk.io/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
)

const (
	UpgradeName = "v4"

	capabilityStoreKey = "capability"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			// new module added in v4
		},
		Deleted: []string{
			capabilityStoreKey,
		},
	},
}
