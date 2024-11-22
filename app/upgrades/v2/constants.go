package v2

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

const (
	UpgradeName = "v2"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			// new module added in v2
			photontypes.ModuleName,
		},
	},
}
