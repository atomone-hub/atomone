package v4

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	sdkgov "github.com/cosmos/cosmos-sdk/x/gov/types"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/atomone-hub/atomone/app/keepers"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v4
// This versions contains the upgrade to Cosmos SDK v0.50 and IBC v10
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		if err := initGovParams(ctx, keepers.GovKeeper); err != nil {
			return vm, err
		}

		return vm, nil
	}
}

// initGovParams initializes the missing gov modules added in AtomOne SDK v0.50.
func initGovParams(ctx context.Context, govKeeper *govkeeper.Keeper) error {
	params, err := govKeeper.Params.Get(ctx)
	if err != nil {
		return err
	}

	defaultParams := sdkgovv1.DefaultParams()
	params.ProposalCancelRatio = defaultParams.ProposalCancelRatio
	params.ProposalCancelDest = authtypes.NewModuleAddress(sdkgov.ModuleName).String()
	params.MinDepositRatio = defaultParams.MinDepositRatio
	params.GovernorStatusChangePeriod = defaultParams.GovernorStatusChangePeriod
	params.MinGovernorSelfDelegation = math.NewInt(10000_000000).String() // to be elegible as governor must have 10K ATONE staked

	if err := govKeeper.Params.Set(ctx, params); err != nil {
		return fmt.Errorf("failed to set gov params: %w", err)
	}

	return nil
}
