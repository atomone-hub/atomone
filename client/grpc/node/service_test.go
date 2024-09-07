package node

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/client"
	sdk "github.com/atomone-hub/atomone/types"
)

func TestServiceServer_Config(t *testing.T) {
	svr := NewQueryServer(client.Context{})
	ctx := sdk.Context{}.WithMinGasPrices(sdk.NewDecCoins(sdk.NewInt64DecCoin("stake", 15)))
	goCtx := sdk.WrapSDKContext(ctx)

	resp, err := svr.Config(goCtx, &ConfigRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, ctx.MinGasPrices().String(), resp.MinimumGasPrice)
}
