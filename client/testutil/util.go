package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/atomone-hub/atomone/runtime"

	runtimev1alpha1 "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	"cosmossdk.io/depinject"

	"github.com/atomone-hub/atomone/codec"
	"github.com/atomone-hub/atomone/core/appconfig"
)

var TestConfig = appconfig.Compose(&appv1alpha1.Config{
	Modules: []*appv1alpha1.ModuleConfig{
		{
			Name: "runtime",
			Config: appconfig.WrapAny(&runtimev1alpha1.Module{
				AppName: "clientTest",
			}),
		},
	},
})

func MakeTestCodec(t *testing.T) codec.Codec { //nolint: thelper
	var cdc codec.Codec
	err := depinject.Inject(TestConfig, &cdc)
	require.NoError(t, err)
	return cdc
}
