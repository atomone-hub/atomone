package testutil

import (
	_ "github.com/atomone-hub/atomone/x/auth"
	_ "github.com/atomone-hub/atomone/x/auth/tx/config"
	_ "github.com/atomone-hub/atomone/x/bank"
	_ "github.com/atomone-hub/atomone/x/capability"
	_ "github.com/atomone-hub/atomone/x/consensus"
	_ "github.com/atomone-hub/atomone/x/genutil"
	_ "github.com/atomone-hub/atomone/x/params"
	_ "github.com/atomone-hub/atomone/x/staking"

	runtimev1alpha1 "github.com/atomone-hub/atomone/api/atomone/app/runtime/v1alpha1"
	appv1alpha1 "github.com/atomone-hub/atomone/api/atomone/app/v1alpha1"
	authmodulev1 "github.com/atomone-hub/atomone/api/atomone/auth/module/v1"
	bankmodulev1 "github.com/atomone-hub/atomone/api/atomone/bank/module/v1"
	capabilitymodulev1 "github.com/atomone-hub/atomone/api/atomone/capability/module/v1"
	consensusmodulev1 "github.com/atomone-hub/atomone/api/atomone/consensus/module/v1"
	genutilmodulev1 "github.com/atomone-hub/atomone/api/atomone/genutil/module/v1"
	paramsmodulev1 "github.com/atomone-hub/atomone/api/atomone/params/module/v1"
	stakingmodulev1 "github.com/atomone-hub/atomone/api/atomone/staking/module/v1"
	txconfigv1 "github.com/atomone-hub/atomone/api/atomone/tx/config/v1"

	"github.com/atomone-hub/atomone/core/appconfig"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	banktypes "github.com/atomone-hub/atomone/x/bank/types"
	capabilitytypes "github.com/atomone-hub/atomone/x/capability/types"
	consensustypes "github.com/atomone-hub/atomone/x/consensus/types"
	genutiltypes "github.com/atomone-hub/atomone/x/genutil/types"
	paramstypes "github.com/atomone-hub/atomone/x/params/types"
	stakingtypes "github.com/atomone-hub/atomone/x/staking/types"
)

var AppConfig = appconfig.Compose(&appv1alpha1.Config{
	Modules: []*appv1alpha1.ModuleConfig{
		{
			Name: "runtime",
			Config: appconfig.WrapAny(&runtimev1alpha1.Module{
				AppName: "CapabilityApp",
				BeginBlockers: []string{
					capabilitytypes.ModuleName,
					stakingtypes.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					genutiltypes.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
				EndBlockers: []string{
					stakingtypes.ModuleName,
					capabilitytypes.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					genutiltypes.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
				InitGenesis: []string{
					capabilitytypes.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					stakingtypes.ModuleName,
					genutiltypes.ModuleName,
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
					{Account: stakingtypes.BondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
					{Account: stakingtypes.NotBondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
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
			Name: capabilitytypes.ModuleName,
			Config: appconfig.WrapAny(&capabilitymodulev1.Module{
				SealKeeper: true,
			}),
		},
	},
})
