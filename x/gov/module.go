package gov

// DONTCOVER

import (
	"context"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/gov"

	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	sdkgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// AppModuleBasic defines the basic application module used by the gov module.
type AppModuleBasic struct {
	gov.AppModuleBasic
}

// NewAppModuleBasic creates a new AppModuleBasic object
func NewAppModuleBasic(appModuleBasic gov.AppModuleBasic) AppModuleBasic {
	return AppModuleBasic{appModuleBasic}
}

// RegisterLegacyAminoCodec registers the gov module's types for the given codec.
func (am AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	am.AppModuleBasic.RegisterLegacyAminoCodec(cdc)

	// forked types
	v1beta1.RegisterLegacyAminoCodec(cdc)
	v1.RegisterLegacyAminoCodec(cdc)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the gov module.
func (a AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	a.AppModuleBasic.RegisterGRPCGatewayRoutes(clientCtx, mux)

	if err := v1.RegisterQueryHandlerClient(context.Background(), mux, v1.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}

	if err := v1beta1.RegisterQueryHandlerClient(context.Background(), mux, v1beta1.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces implements InterfaceModule.RegisterInterfaces
func (a AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	a.AppModuleBasic.RegisterInterfaces(registry)

	v1.RegisterInterfaces(registry)
	v1beta1.RegisterInterfaces(registry)
}

// AppModule implements an application module for the gov module.
type AppModule struct {
	AppModuleBasic
	gov.AppModule
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	appModule gov.AppModule,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{appModule.AppModuleBasic},
		AppModule:      appModule,
	}
}

var _ appmodule.AppModule = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// Name returns the gov module's name.
func (AppModule) Name() string {
	return sdkgovtypes.ModuleName
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	am.RegisterServices(cfg)

	msgServer := keeper.NewMsgServerImpl(am.keeper)
	v1beta1.RegisterMsgServer(cfg.MsgServer(), keeper.NewLegacyMsgServerImpl(am.accountKeeper.GetModuleAddress(govtypes.ModuleName).String(), msgServer))
	v1.RegisterMsgServer(cfg.MsgServer(), msgServer)

	legacyQueryServer := keeper.NewLegacyQueryServer(am.AppModule.Keeper)
	v1beta1.RegisterQueryServer(cfg.QueryServer(), legacyQueryServer)
	v1.RegisterQueryServer(cfg.QueryServer(), am.AppModule)
}
