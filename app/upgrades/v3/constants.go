package v3

import (
	store "cosmossdk.io/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
)

const (
	UpgradeName = "v3"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			// new module added in v3
		},
		Deleted: []string{},
	},
}
