package v4_1

import (
	store "cosmossdk.io/store/types"

	"github.com/atomone-hub/atomone/app/upgrades"
)

const (
	// UpgradeName is the on-chain name of the testnet-only v4.1 upgrade.
	UpgradeName = "v4.1"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	// No store changes: this upgrade only initializes the staking
	// KeyRotationFee param missed by the testnet's v4 run.
	StoreUpgrades: store.StoreUpgrades{},
}
