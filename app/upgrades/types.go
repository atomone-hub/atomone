package upgrades

import (
	"context"
	"fmt"

	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/atomone-hub/atomone/app/keepers"
)

// AppendAllowedClient appends a light client type to the IBC 02-client
// AllowedClients param so clients of that type can be created. It is a
// no-op when the type is already allowed, including via the allow-all
// wildcard.
func AppendAllowedClient(ctx context.Context, ibcKeeper *ibckeeper.Keeper, clientType string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := ibcKeeper.ClientKeeper.GetParams(sdkCtx)
	if params.IsAllowedClient(clientType) {
		return nil
	}
	params.AllowedClients = append(params.AllowedClients, clientType)
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to validate ibc client params: %w", err)
	}
	ibcKeeper.ClientKeeper.SetParams(sdkCtx, params)
	return nil
}

// Upgrade defines a struct containing necessary fields that a SoftwareUpgradeProposal
// must have written, in order for the state migration to go smoothly.
// An upgrade must implement this struct, and then set it in the app.go.
// The app.go will then define the handler.
type Upgrade struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string

	// CreateUpgradeHandler defines the function that creates an upgrade handler
	CreateUpgradeHandler func(*module.Manager, codec.Codec, module.Configurator, *keepers.AppKeepers) upgradetypes.UpgradeHandler

	// Store upgrades, should be used for any new modules introduced, new modules deleted, or store names renamed.
	StoreUpgrades store.StoreUpgrades
}

// Fork defines a struct containing the requisite fields for a non-software upgrade proposal
// Hard Fork at a given height to implement.
// There is one time code that can be added at the start of the Fork, in `BeginForkLogic`.
// Any other change in the code should be height-gated, if the goal is to have old and new binaries
// to be compatible prior to the upgrade height.
type Fork struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string
	// height the upgrade occurs at
	UpgradeHeight int64

	// Function that runs some custom state transition code at the beginning of a fork.
	BeginForkLogic func(ctx sdk.Context, keepers *keepers.AppKeepers)
}
