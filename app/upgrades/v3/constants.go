package v3

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
	dynamicfeetypes "github.com/atomone-hub/atomone/x/dynamicfee/types"
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
			dynamicfeetypes.ModuleName,
		},
	},
}
