package keeper_test

import (
	"context"
	"testing"

	"github.com/atomone-hub/atomone/x/photon/keeper"
	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t *testing.T) (types.MsgServer, context.Context) {
	k, _, ctx := testutil.SetupPhotonKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}

func TestMsgServer(t *testing.T) {
	ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
}
