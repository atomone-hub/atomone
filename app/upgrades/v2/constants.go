package v2

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"

	"github.com/atomone-hub/atomone/app/upgrades"
	feemarkettypes "github.com/atomone-hub/atomone/x/feemarket/types"
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
			// new modules added in v2
			photontypes.ModuleName,
			feemarkettypes.ModuleName,
		},
		Deleted: []string{
			crisistypes.ModuleName,
		},
	},
}
