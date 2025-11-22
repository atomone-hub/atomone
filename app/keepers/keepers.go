package keepers

import (
	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	transferv2 "github.com/cosmos/ibc-go/v10/modules/apps/transfer/v2"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradecompat "cosmossdk.io/x/upgrade"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	coredaoskeeper "github.com/atomone-hub/atomone/x/coredaos/keeper"
	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	dynamicfeekeeper "github.com/atomone-hub/atomone/x/dynamicfee/keeper"
	dynamicfeetypes "github.com/atomone-hub/atomone/x/dynamicfee/types"
	govkeeper "github.com/atomone-hub/atomone/x/gov/keeper"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	govv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	photonkeeper "github.com/atomone-hub/atomone/x/photon/keeper"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

type AppKeepers struct {
	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper  authkeeper.AccountKeeper
	BankKeeper     bankkeeper.Keeper
	StakingKeeper  *stakingkeeper.Keeper
	SlashingKeeper slashingkeeper.Keeper
	MintKeeper     mintkeeper.Keeper
	DistrKeeper    distrkeeper.Keeper
	GovKeeper      *govkeeper.Keeper
	UpgradeKeeper  *upgradekeeper.Keeper
	ParamsKeeper   paramskeeper.Keeper
	// IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCKeeper             *ibckeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper
	PhotonKeeper          *photonkeeper.Keeper
	DynamicfeeKeeper      *dynamicfeekeeper.Keeper
	CoreDaosKeeper        *coredaoskeeper.Keeper

	// Modules
	ICAModule      ica.AppModule
	TransferModule transfer.AppModule
	TMClientModule ibctm.AppModule
}

func NewAppKeeper(
	appCodec codec.Codec,
	bApp *baseapp.BaseApp,
	legacyAmino *codec.LegacyAmino,
	maccPerms map[string][]string,
	modAccAddrs map[string]bool,
	blockedAddress map[string]bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	logger log.Logger,
	appOpts servertypes.AppOptions,
) AppKeepers {
	authorityStr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	appKeepers := AppKeepers{}
	// Set keys KVStoreKey, TransientStoreKey, MemoryStoreKey
	appKeepers.GenerateKeys()

	appKeepers.ParamsKeeper = initParamsKeeper(
		appCodec,
		legacyAmino,
		appKeepers.keys[paramstypes.StoreKey],
		appKeepers.tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	appKeepers.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[consensusparamtypes.StoreKey]),
		authorityStr,
		runtime.EventService{},
	)
	bApp.SetParamStore(&appKeepers.ConsensusParamsKeeper.ParamsStore)
	bApp.SetVersionModifier(consensus.ProvideAppVersionModifier(appKeepers.ConsensusParamsKeeper))

	// Add normal keepers
	appKeepers.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addressCodec,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authorityStr,
	)

	appKeepers.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[banktypes.StoreKey]),
		appKeepers.AccountKeeper,
		blockedAddress,
		authorityStr,
		logger,
	)

	appKeepers.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(appKeepers.keys[authzkeeper.StoreKey]),
		appCodec,
		bApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
	)

	appKeepers.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[feegrant.StoreKey]),
		appKeepers.AccountKeeper,
	)

	appKeepers.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[stakingtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authorityStr,
		appCodec.InterfaceRegistry().SigningContext().ValidatorAddressCodec(),
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	appKeepers.PhotonKeeper = photonkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[photontypes.StoreKey],
		authorityStr,
		appKeepers.BankKeeper,
		appKeepers.AccountKeeper,
		appKeepers.StakingKeeper,
	)

	appKeepers.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[minttypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authorityStr,
	)

	appKeepers.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[distrtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		authtypes.FeeCollectorName,
		authorityStr,
	)

	appKeepers.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(appKeepers.keys[slashingtypes.StoreKey]),
		appKeepers.StakingKeeper,
		authorityStr,
	)

	// UpgradeKeeper must be created before IBCKeeper
	appKeepers.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(appKeepers.keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		bApp,
		authorityStr,
	)

	appKeepers.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibcexported.StoreKey]),
		appKeepers.GetSubspace(ibcexported.ModuleName),
		upgradecompat.NewUpgradeKeeperCompat(appKeepers.UpgradeKeeper),
		authorityStr,
	)

	// provider depends on gov, so gov must be registered first
	govConfig := govtypes.DefaultConfig()
	// set the MaxMetadataLen for proposals to the same value as it was pre-sdk v0.47.x
	govConfig.MaxMetadataLen = 10200
	appKeepers.GovKeeper = govkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[govtypes.StoreKey],
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		bApp.MsgServiceRouter(),
		govConfig,
		authorityStr,
	)

	// Set legacy router for backwards compatibility with gov v1beta1
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, func(ctx sdk.Context, content govv1beta1.Content) error {
			return params.NewParamChangeProposalHandler(appKeepers.ParamsKeeper)(ctx, content)
		})
	appKeepers.GovKeeper.SetLegacyRouter(govRouter)

	appKeepers.CoreDaosKeeper = coredaoskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[coredaostypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		appKeepers.GovKeeper,
		appKeepers.StakingKeeper,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	appKeepers.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			appKeepers.DistrKeeper.Hooks(),
			appKeepers.SlashingKeeper.Hooks(),
			appKeepers.CoreDaosKeeper.StakingHooks(),
		),
	)

	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[evidencetypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.SlashingKeeper,
		appKeepers.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	appKeepers.EvidenceKeeper = *evidenceKeeper

	// ICA Host keeper
	appKeepers.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[icahosttypes.StoreKey]),
		appKeepers.GetSubspace(icahosttypes.SubModuleName),
		appKeepers.IBCKeeper.ChannelKeeper, // ICS4Wrapper
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.AccountKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		authorityStr,
	)

	appKeepers.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibctransfertypes.StoreKey]),
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		bApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authorityStr,
	)

	appKeepers.DynamicfeeKeeper = dynamicfeekeeper.NewKeeper(
		appCodec,
		appKeepers.keys[dynamicfeetypes.StoreKey],
		appKeepers.PhotonKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Middleware Stacks
	appKeepers.ICAModule = ica.NewAppModule(nil, &appKeepers.ICAHostKeeper)
	appKeepers.TransferModule = transfer.NewAppModule(appKeepers.TransferKeeper)

	// create IBC module from bottom to top of stack
	var (
		transferStack   porttypes.IBCModule = transfer.NewIBCModule(appKeepers.TransferKeeper)
		transferStackV2 ibcapi.IBCModule    = transferv2.NewIBCModule(appKeepers.TransferKeeper)
	)

	// Add transfer stack to IBC Router

	// Create Interchain Accounts Stack
	var icaHostStack porttypes.IBCModule = icahost.NewIBCModule(appKeepers.ICAHostKeeper)

	// Create IBC Router & seal
	ibcRouter := porttypes.NewRouter().
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack)

	ibcv2Router := ibcapi.NewRouter().
		AddRoute(ibctransfertypes.PortID, transferStackV2)

	appKeepers.IBCKeeper.SetRouter(ibcRouter)
	appKeepers.IBCKeeper.SetRouterV2(ibcv2Router)

	// Light Clients
	clientKeeper := appKeepers.IBCKeeper.ClientKeeper
	storeProvider := clientKeeper.GetStoreProvider()

	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	appKeepers.IBCKeeper.ClientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

	appKeepers.TMClientModule = ibctm.NewAppModule(tmLightClientModule)

	return appKeepers
}

// GetSubspace returns a param subspace for a given module name.
func (appKeepers *AppKeepers) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, ok := appKeepers.ParamsKeeper.GetSubspace(moduleName)
	if !ok {
		panic("couldn't load subspace for module: " + moduleName)
	}
	return subspace
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	//nolint: staticcheck // SA1019: moduletypes.ParamKeyTable is deprecated
	paramsKeeper.Subspace(authtypes.ModuleName).WithKeyTable(authtypes.ParamKeyTable())
	paramsKeeper.Subspace(stakingtypes.ModuleName).WithKeyTable(stakingtypes.ParamKeyTable())   //nolint:staticcheck // SA1019
	paramsKeeper.Subspace(banktypes.ModuleName).WithKeyTable(banktypes.ParamKeyTable())         //nolint:staticcheck // SA1019
	paramsKeeper.Subspace(minttypes.ModuleName).WithKeyTable(minttypes.ParamKeyTable())         //nolint:staticcheck // SA1019
	paramsKeeper.Subspace(distrtypes.ModuleName).WithKeyTable(distrtypes.ParamKeyTable())       //nolint:staticcheck // SA1019
	paramsKeeper.Subspace(slashingtypes.ModuleName).WithKeyTable(slashingtypes.ParamKeyTable()) //nolint:staticcheck // SA1019
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())              //nolint:staticcheck // SA1019
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)

	return paramsKeeper
}
