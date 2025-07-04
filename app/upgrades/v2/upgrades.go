package v2

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/atomone-hub/atomone/app/keepers"
	photonkeeper "github.com/atomone-hub/atomone/x/photon/keeper"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v2
// which executes the following migrations:
//   - add new denom metadata for photon in the bank module store.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		sdkCtx.Logger().Info("Starting module migrations...")
		// RunMigrations will detect the add of the photon module, will initiate
		// its genesis and will fill the versionMap with its consensus version.
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		// Set photon.params.txFeeExceptions to '*' to allow transactions to use
		// any fee. This is a temporary measure to give users time to migrate to
		// the new fee model.
		if err := setPhotonTxFeeExceptions(sdkCtx, keepers.PhotonKeeper); err != nil {
			return vm, err
		}
		// Add the photon denom metadata to the bank module store
		setPhotonDenomMetadata(sdkCtx, keepers.BankKeeper)
		sdkCtx.Logger().Info("Upgrade complete")
		return vm, nil
	}
}

func setPhotonTxFeeExceptions(ctx sdk.Context, k *photonkeeper.Keeper) error {
	ctx.Logger().Info("Setting photon.params.txFeeExceptions to '*'...")
	params := k.GetParams(ctx)
	params.TxFeeExceptions = []string{"*"}
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}
	ctx.Logger().Info("photon.params.txFeeExceptions to '*' set")
	return nil
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
