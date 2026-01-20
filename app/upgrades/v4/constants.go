package v4

import (
	store "cosmossdk.io/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
)

const (
	UpgradeName        = "v4"
	capabilityStoreKey = "capability"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			// new module added in v4
			coredaostypes.StoreKey,
			icacontrollertypes.StoreKey,
			// x/gov has been added but it uses the same store key as the x/gov fork from v3
		},
		Deleted: []string{
			capabilityStoreKey,
		},
	},
}
