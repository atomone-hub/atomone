package testutil

import (
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	"cosmossdk.io/core/appconfig"

	"github.com/cosmos/cosmos-sdk/testutil/configurator"

	govmodulev1 "github.com/atomone-hub/atomone/x/gov/types/module"
)

func GovModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.ModuleConfigs["gov"] = &appv1alpha1.ModuleConfig{
			Name:   "gov",
			Config: appconfig.WrapAny(&govmodulev1.Module{}),
		}
	}
}
