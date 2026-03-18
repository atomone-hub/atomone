package v4

import (
	"testing"

	"cosmossdk.io/math"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
)

func TestSetNewGovParamDefaultsUsesSDKDefaults(t *testing.T) {
	var params sdkgovv1.Params

	setNewGovParamDefaults(&params)

	defaultParams := sdkgovv1.DefaultParams()
	require.Equal(t, defaultParams.ProposalCancelRatio, params.ProposalCancelRatio)
	require.Equal(t, defaultParams.ProposalCancelDest, params.ProposalCancelDest)
	require.Equal(t, defaultParams.MinDepositRatio, params.MinDepositRatio)
	require.Equal(t, defaultParams.GovernorStatusChangePeriod, params.GovernorStatusChangePeriod)
	require.Equal(t, math.NewInt(10000_000000).String(), params.MinGovernorSelfDelegation)
}
