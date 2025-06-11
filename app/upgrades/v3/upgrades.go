package v3

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/atomone-hub/atomone/app/keepers"
	govkeeper "github.com/atomone-hub/atomone/x/gov/keeper"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v3
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Starting module migrations...")
		// RunMigrations will detect the add of the feemarket module, will initiate
		// its genesis and will fill the versionMap with its consensus version.
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		if err := initGovDynamicQuorum(ctx, keepers.GovKeeper); err != nil {
			return vm, err
		}
		ctx.Logger().Info("Upgrade complete")
		return vm, nil
	}
}

// initGovDynamicQuorum initialized the gov module for the dynamic quorum
// features, which means setting the new parameters min/max quorums and the
// participation ema.
func initGovDynamicQuorum(ctx sdk.Context, govKeeper *govkeeper.Keeper) error {
	ctx.Logger().Info("Initializing gov module for dynamic quorum...")
	params := govKeeper.GetParams(ctx)
	defaultParams := v1.DefaultParams()
	params.MinQuorum = defaultParams.MinQuorum
	params.MaxQuorum = defaultParams.MaxQuorum
	params.MinConstitutionAmendmentQuorum = defaultParams.MinConstitutionAmendmentQuorum
	params.MaxConstitutionAmendmentQuorum = defaultParams.MaxConstitutionAmendmentQuorum
	params.MinLawQuorum = defaultParams.MinLawQuorum
	params.MaxLawQuorum = defaultParams.MaxLawQuorum
	if err := govKeeper.SetParams(ctx, params); err != nil {
		return fmt.Errorf("set gov params: %w", err)
	}
	// NOTE(tb): Disregarding whales' votes, the current participation is less than 12%
	initParticipationEma := sdk.NewDecWithPrec(12, 2)
	govKeeper.SetParticipationEMA(ctx, initParticipationEma)
	govKeeper.SetConstitutionAmendmentParticipationEMA(ctx, initParticipationEma)
	govKeeper.SetLawParticipationEMA(ctx, initParticipationEma)
	ctx.Logger().Info("Gov module initialized for dynamic quorum")
	return nil
}
