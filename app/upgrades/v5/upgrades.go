package v5

import (
	"context"
	"fmt"

	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/atomone-hub/atomone/app/keepers"
	"github.com/atomone-hub/atomone/modules/spectre"
)

// CreateUpgradeHandler returns the upgrade handler for AtomOne v5.
// This version registers the spectre Ethereum light client.
func CreateUpgradeHandler(
	mm *module.Manager,
	cdc codec.Codec,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		if err := AddSpectreToIBCAllowedClients(ctx, keepers.IBCKeeper); err != nil {
			return vm, err
		}

		return vm, nil
	}
}

// AddSpectreToIBCAllowedClients appends the spectre light client type to the
// IBC 02-client AllowedClients param so spectre clients can be created. It is
// a no-op when spectre is already allowed, including via the allow-all
// wildcard.
func AddSpectreToIBCAllowedClients(ctx context.Context, ibcKeeper *ibckeeper.Keeper) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := ibcKeeper.ClientKeeper.GetParams(sdkCtx)
	if params.IsAllowedClient(spectre.ModuleName) {
		return nil
	}
	params.AllowedClients = append(params.AllowedClients, spectre.ModuleName)
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to validate ibc client params: %w", err)
	}
	ibcKeeper.ClientKeeper.SetParams(sdkCtx, params)
	return nil
}
