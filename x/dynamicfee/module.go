package dynamicfee

import (
	"context"
	"encoding/json"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	store "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	modulev1 "github.com/atomone-hub/atomone/api/atomone/dynamicfee/module/v1"
	"github.com/atomone-hub/atomone/x/dynamicfee/client/cli"
	"github.com/atomone-hub/atomone/x/dynamicfee/keeper"
	"github.com/atomone-hub/atomone/x/dynamicfee/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

// ConsensusVersion is the x/dynamicfee module's consensus version identifier.
const ConsensusVersion = 1

var (
	_ module.AppModuleBasic = AppModule{}
	_ module.HasGenesis     = AppModule{}
	_ module.HasServices    = AppModule{}

	_ appmodule.AppModule     = AppModule{}
	_ appmodule.HasEndBlocker = AppModule{}
)

// AppModuleBasic defines the base interface that the x/dynamicfee module exposes to the application.
type AppModuleBasic struct {
	cdc codec.Codec
}

// Name returns the name of x/dynamicfee module.
func (amb AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers the necessary types from the x/dynamicfee module for amino
// serialization.
func (amb AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the necessary implementations / interfaces in the x/dynamicfee
// module w/ the interface-registry.
func (amb AppModuleBasic) RegisterInterfaces(ir codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(ir)
}

// RegisterGRPCGatewayRoutes registers the necessary REST routes for the GRPC-gateway to
// the x/dynamicfee module QueryService on mux. This method panics on failure.
func (amb AppModuleBasic) RegisterGRPCGatewayRoutes(cliCtx client.Context, mux *runtime.ServeMux) {
	// Register the gate-way routes w/ the provided mux.
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(cliCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd is a no-op, as no txs are registered for submission (apart from messages that
// can only be executed by governance).
func (amb AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd returns the x/dynamicfee module base query cli-command.
func (amb AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule represents an application module for the x/dynamicfee module.
type AppModule struct {
	AppModuleBasic

	k keeper.Keeper
}

// NewAppModule returns an application module for the x/dynamicfee module.
func NewAppModule(cdc codec.Codec, k keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{
			cdc: cdc,
		},
		k: k,
	}
}

// EndBlock returns an endblocker for the x/feemarket module.
func (am AppModule) EndBlock(ctx context.Context) error {
	return am.k.EndBlock(ctx)
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// RegisterServices registers the module's services with the app's module configurator.
func (am AppModule) RegisterServices(cfc module.Configurator) {
	types.RegisterMsgServer(cfc.MsgServer(), keeper.NewMsgServer(&am.k))
	types.RegisterQueryServer(cfc.QueryServer(), keeper.NewQueryServer(am.k))
}

// DefaultGenesis returns default genesis state as raw bytes for the dynamicfee
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultAIMDGenesisState())
}

// ValidateGenesis performs genesis state validation for the dynamicfee module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return err
	}

	return gs.ValidateBasic()
}

// InitGenesis performs the genesis initialization for the x/dynamicfee module. This method returns
// no validator set updates. This method panics on any errors.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, bz json.RawMessage) {
	var gs types.GenesisState
	am.cdc.MustUnmarshalJSON(bz, &gs)
	am.k.InitGenesis(ctx, gs)
}

// ExportGenesis returns the dynamicfee module's exported genesis state as raw
// JSON bytes. This method panics on any error.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.k.ExportGenesis(ctx)
	return am.cdc.MustMarshalJSON(gs)
}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type Inputs struct {
	depinject.In

	Config *modulev1.Module
	Cdc    codec.Codec
	Key    *store.KVStoreKey
}

type Outputs struct {
	depinject.Out

	Keeper keeper.Keeper
	Module appmodule.AppModule
}

func ProvideModule(in Inputs) Outputs {
	var (
		authority sdk.AccAddress
		err       error
	)
	if in.Config.Authority != "" {
		authority, err = sdk.AccAddressFromBech32(in.Config.Authority)
		if err != nil {
			panic(err)
		}
	} else {
		authority = authtypes.NewModuleAddress(govtypes.ModuleName)
	}

	Keeper := keeper.NewKeeper(
		in.Cdc,
		in.Key,
		nil,
		authority.String(),
	)

	m := NewAppModule(in.Cdc, *Keeper)

	return Outputs{Keeper: *Keeper, Module: m}
}
