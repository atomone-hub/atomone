package v4

import (
	store "cosmossdk.io/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
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
			coredaostypes.ModuleName,
		},
		Deleted: []string{
			capabilityStoreKey,
		},
	},
}
