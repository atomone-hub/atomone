package v1

import (
	"testing"

	sdkv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
)

func TestConvertAtomOneParamsToSDKUsesSDKCancelDefaults(t *testing.T) {
	params := DefaultParams()

	sdkParams := ConvertAtomOneParamsToSDK(&params)
	defaultParams := sdkv1.DefaultParams()

	require.Equal(t, defaultParams.ProposalCancelRatio, sdkParams.ProposalCancelRatio)
	require.Equal(t, defaultParams.ProposalCancelDest, sdkParams.ProposalCancelDest)
	require.NoError(t, sdkParams.ValidateBasic())
}
