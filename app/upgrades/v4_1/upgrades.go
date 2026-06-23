package v4_1

import (
	"context"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/atomone-hub/atomone/app/keepers"
)

// CreateUpgradeHandler returns the upgrade handler for AtomOne v4.1.
//
// This is a testnet-only upgrade. The testnet ran the v4 upgrade with a binary
// (v4.0.0-rc) that predated the consensus pubkey rotation feature, so its v4
// handler never initialized the staking KeyRotationFee param. Mainnet is
// unaffected: it runs the v4.0.0 binary whose MigrateStakingParams sets the
// param during the v4 upgrade. After this upgrade the testnet reaches the same
// post-v4 state as mainnet.
//
// No module consensus version changed between the two binaries, so there are no
// pending module migrations and RunMigrations is intentionally not called.
func CreateUpgradeHandler(
	_ *module.Manager,
	_ codec.Codec,
	_ module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		if err := InitKeyRotationFee(ctx, keepers.StakingKeeper); err != nil {
			return vm, err
		}
		return vm, nil
	}
}

// InitKeyRotationFee initializes the staking KeyRotationFee param to 100 ATONE,
// mirroring the value set by v4.MigrateStakingParams so the testnet converges to
// the same state as mainnet.
func InitKeyRotationFee(ctx context.Context, stakingKeeper *stakingkeeper.Keeper) error {
	params, err := stakingKeeper.GetParams(ctx)
	if err != nil {
		return err
	}

	// Initialize the consensus pubkey rotation fee to 100 ATONEs
	params.KeyRotationFee = sdk.NewCoin(params.BondDenom, math.NewInt(100_000000))

	return stakingKeeper.SetParams(ctx, params)
}
