package spectre

import (
	"context"
	"encoding/json"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var (
	_ module.AppModuleBasic = (*AppModuleBasic)(nil)
	_ appmodule.AppModule   = (*AppModule)(nil)

	_ appmodule.HasEndBlocker = (*AppModule)(nil)
)

// Spectre is the client type of this module.
const Spectre = ModuleName

// AppModuleBasic defines the basic application module used by the spectre
// light client. Only the RegisterInterfaces function needs to be
// implemented. All other functions perform a no-op.
type AppModuleBasic struct{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModuleBasic) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModuleBasic) IsAppModule() {}

// Name returns the spectre module name.
func (AppModuleBasic) Name() string {
	return ModuleName
}

// RegisterLegacyAminoCodec performs a no-op. The spectre client does not
// support amino.
func (AppModuleBasic) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

// RegisterInterfaces registers module concrete types into protobuf Any. This
// allows core IBC to unmarshal spectre light client types.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	RegisterInterfaces(registry)
}

// DefaultGenesis performs a no-op. Genesis is not supported for the spectre
// light client.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ValidateGenesis performs a no-op. Genesis is not supported for the spectre
// light client.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	return nil
}

// RegisterGRPCGatewayRoutes performs a no-op.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

// GetTxCmd performs a no-op. Please see the 02-client cli commands.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd performs a no-op. Please see the 02-client cli commands.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

// ClientKeeper is the subset of the IBC 02-client keeper the spectre module
// needs: iterating its clients and opening their stores.
type ClientKeeper interface {
	IterateClientStates(ctx sdk.Context, prefix []byte, cb func(clientID string, state exported.ClientState) bool)
	ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore
}

// AppModule is the application module for the spectre client module.
type AppModule struct {
	AppModuleBasic
	lightClientModule LightClientModule
	cdc               codec.BinaryCodec
	clientKeeper      ClientKeeper
}

// NewAppModule creates a new spectre client module.
func NewAppModule(cdc codec.BinaryCodec, lightClientModule LightClientModule, clientKeeper ClientKeeper) AppModule {
	return AppModule{
		lightClientModule: lightClientModule,
		cdc:               cdc,
		clientKeeper:      clientKeeper,
	}
}

// EndBlock pruning knobs. Pruning runs on every pruneIntervalBlocks-th
// block and deletes at most pruneBatchLimit states per client per run, so
// the worst-case per-block work stays bounded while states accumulate far
// slower (one per Ethereum finalization) than the batch limit.
const (
	pruneIntervalBlocks int64 = 100
	pruneBatchLimit     int   = 1000
)

// EndBlock prunes spectre consensus states below the pruning window of
// PruneKeepDuration. Membership verification already rejects heights below
// the window, so a running client loses nothing provable. A failing client
// is logged and skipped: pruning is housekeeping and must not halt the
// chain.
func (am AppModule) EndBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if ctx.BlockHeight()%pruneIntervalBlocks != 0 {
		return nil
	}

	am.clientKeeper.IterateClientStates(ctx, []byte(ModuleName), func(clientID string, state exported.ClientState) bool {
		clientState, ok := state.(*ClientState)
		if !ok {
			return false
		}
		pruned, err := PruneConsensusStates(am.clientKeeper.ClientStore(ctx, clientID), am.cdc, clientState.PruneKeepSlots(), pruneBatchLimit)
		if err != nil {
			ctx.Logger().Error("failed to prune spectre consensus states", "client-id", clientID, "error", err)
			return false
		}
		if pruned > 0 {
			ctx.Logger().Info("pruned spectre consensus states", "client-id", clientID, "count", pruned)
		}
		return false
	})
	return nil
}
