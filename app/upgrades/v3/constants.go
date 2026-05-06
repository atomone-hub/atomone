package v3

import (
	storetypes "cosmossdk.io/store/types"

	dynamicfeetypes "github.com/cosmos/cosmos-sdk/x/dynamicfee/types"

	"github.com/atomone-hub/atomone/app/upgrades"
)

const (
	UpgradeName = "v3"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added: []string{
			// new module added in v3
			dynamicfeetypes.ModuleName,
		},
	},
}
