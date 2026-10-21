package v5

import (
	store "cosmossdk.io/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
)

const UpgradeName = "v5"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
