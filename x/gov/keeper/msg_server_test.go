package keeper

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestMergeLegacyParamsWithSDKParamsPreservesSDKOnlyFields(t *testing.T) {
	current := sdkv1.DefaultParams()
	current.ProposalCancelRatio = "0.42"
	current.ProposalCancelDest = authtypes.NewModuleAddress("distribution").String()
	legacy := v1.DefaultParams()

	merged := mergeLegacyParamsWithSDKParams(current, &legacy)

	require.Equal(t, current.ProposalCancelRatio, merged.ProposalCancelRatio)
	require.Equal(t, current.ProposalCancelDest, merged.ProposalCancelDest)
	require.NoError(t, merged.ValidateBasic())
}
