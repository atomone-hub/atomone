package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/atomone-hub/atomone/app/keepers"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v2
// which executes the following migrations:
//   - add new denom metadata for photon in the bank module store.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Starting module migrations...")
		// Admitedly there's no module migration because v2 doesn't upgrade the
		// SDK, but still running it for demo purpose.
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		// Add the photon denom metadata to the bank module store
		setPhotonDenomMetadata(ctx, keepers.BankKeeper)
		ctx.Logger().Info("Upgrade complete")
		return vm, nil
	}
}

func setPhotonDenomMetadata(ctx sdk.Context, bk bankkeeper.Keeper) {
	ctx.Logger().Info("Adding photon denom metadata...")
	bk.SetDenomMetaData(ctx, banktypes.Metadata{
		Base:        "uphoton",
		Display:     "photon",
		Name:        "AtomOne Photon",
		Symbol:      "PHOTON",
		Description: "The fee token of AtomOne Hub",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "uphoton",
				Exponent: 0,
				Aliases: []string{
					"microphoton",
				},
			},
			{
				Denom:    "mphoton",
				Exponent: 3,
				Aliases: []string{
					"milliphoton",
				},
			},
			{
				Denom:    "photon",
				Exponent: 6,
			},
		},
	})
	ctx.Logger().Info("Photon denom metadata added")
}
