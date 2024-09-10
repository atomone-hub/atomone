package testutil

import (
	_ "github.com/atomone-hub/atomone/x/auth"
	_ "github.com/atomone-hub/atomone/x/auth/tx/config"
	_ "github.com/atomone-hub/atomone/x/authz/module"
	_ "github.com/atomone-hub/atomone/x/bank"
	_ "github.com/atomone-hub/atomone/x/consensus"
	_ "github.com/atomone-hub/atomone/x/genutil"
	_ "github.com/atomone-hub/atomone/x/gov"
	_ "github.com/atomone-hub/atomone/x/mint"
	_ "github.com/atomone-hub/atomone/x/params"
	_ "github.com/atomone-hub/atomone/x/staking"

	runtimev1alpha1 "github.com/atomone-hub/atomone/api/atomone/app/runtime/v1alpha1"
	appv1alpha1 "github.com/atomone-hub/atomone/api/atomone/app/v1alpha1"
	authmodulev1 "github.com/atomone-hub/atomone/api/atomone/auth/module/v1"
	authzmodulev1 "github.com/atomone-hub/atomone/api/atomone/authz/module/v1"
	bankmodulev1 "github.com/atomone-hub/atomone/api/atomone/bank/module/v1"
	consensusmodulev1 "github.com/atomone-hub/atomone/api/atomone/consensus/module/v1"
	genutilmodulev1 "github.com/atomone-hub/atomone/api/atomone/genutil/module/v1"
	mintmodulev1 "github.com/atomone-hub/atomone/api/atomone/mint/module/v1"
	paramsmodulev1 "github.com/atomone-hub/atomone/api/atomone/params/module/v1"
	stakingmodulev1 "github.com/atomone-hub/atomone/api/atomone/staking/module/v1"
	txconfigv1 "github.com/atomone-hub/atomone/api/atomone/tx/config/v1"

	"github.com/atomone-hub/atomone/core/appconfig"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	"github.com/atomone-hub/atomone/x/authz"
	banktypes "github.com/atomone-hub/atomone/x/bank/types"
	consensustypes "github.com/atomone-hub/atomone/x/consensus/types"
	genutiltypes "github.com/atomone-hub/atomone/x/genutil/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	minttypes "github.com/atomone-hub/atomone/x/mint/types"
	paramstypes "github.com/atomone-hub/atomone/x/params/types"
	stakingtypes "github.com/atomone-hub/atomone/x/staking/types"
)

var AppConfig = appconfig.Compose(&appv1alpha1.Config{
	Modules: []*appv1alpha1.ModuleConfig{
		{
			Name: "runtime",
			Config: appconfig.WrapAny(&runtimev1alpha1.Module{
				AppName: "AuthzApp",
				BeginBlockers: []string{
					minttypes.ModuleName,
					stakingtypes.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					genutiltypes.ModuleName,
					authz.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
				EndBlockers: []string{
					minttypes.ModuleName,
					stakingtypes.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					genutiltypes.ModuleName,
					authz.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
				InitGenesis: []string{
					authtypes.ModuleName,
					banktypes.ModuleName,
					stakingtypes.ModuleName,
					minttypes.ModuleName,
					genutiltypes.ModuleName,
					authz.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
			}),
		},
		{
			Name: authtypes.ModuleName,
			Config: appconfig.WrapAny(&authmodulev1.Module{
				Bech32Prefix: "cosmos",
				ModuleAccountPermissions: []*authmodulev1.ModuleAccountPermission{
					{Account: authtypes.FeeCollectorName},
					{Account: minttypes.ModuleName, Permissions: []string{authtypes.Minter}},
					{Account: stakingtypes.BondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
					{Account: stakingtypes.NotBondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
					{Account: govtypes.ModuleName, Permissions: []string{authtypes.Burner}},
				},
			}),
		},
		{
			Name:   banktypes.ModuleName,
			Config: appconfig.WrapAny(&bankmodulev1.Module{}),
		},
		{
			Name:   stakingtypes.ModuleName,
			Config: appconfig.WrapAny(&stakingmodulev1.Module{}),
		},
		{
			Name:   paramstypes.ModuleName,
			Config: appconfig.WrapAny(&paramsmodulev1.Module{}),
		},
		{
			Name:   "tx",
			Config: appconfig.WrapAny(&txconfigv1.Config{}),
		},
		{
			Name:   genutiltypes.ModuleName,
			Config: appconfig.WrapAny(&genutilmodulev1.Module{}),
		},
		{
			Name:   consensustypes.ModuleName,
			Config: appconfig.WrapAny(&consensusmodulev1.Module{}),
		},
		{
			Name:   authz.ModuleName,
			Config: appconfig.WrapAny(&authzmodulev1.Module{}),
		},
		{
			Name:   minttypes.ModuleName,
			Config: appconfig.WrapAny(&mintmodulev1.Module{}),
		},
	},
})
