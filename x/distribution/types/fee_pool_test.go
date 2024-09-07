package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/distribution/types"
)

func TestValidateGenesis(t *testing.T) {
	fp := types.InitialFeePool()
	require.Nil(t, fp.ValidateGenesis())

	fp2 := types.FeePool{CommunityPool: sdk.DecCoins{{Denom: "stake", Amount: math.LegacyNewDec(-1)}}}
	require.NotNil(t, fp2.ValidateGenesis())
}
