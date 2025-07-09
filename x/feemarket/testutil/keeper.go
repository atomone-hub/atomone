package testutil

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtime "github.com/cometbft/cometbft/types/time"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/feemarket/keeper"
	"github.com/atomone-hub/atomone/x/feemarket/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

func SetupMsgServer(t *testing.T, maxBlockGas uint64) (types.MsgServer, *keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := SetupKeeper(t, maxBlockGas)
	return keeper.NewMsgServer(k), k, ctx
}

func SetupQueryServer(t *testing.T, maxBlockGas uint64) (types.QueryServer, *keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := SetupKeeper(t, maxBlockGas)
	return keeper.NewQueryServer(*k), k, ctx
}

const MaxBlockGas = 30_000_000

func SetupKeeper(t *testing.T, maxBlockGas uint64) (*keeper.Keeper, sdk.Context) {
	t.Helper()

	key := sdk.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, sdk.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(tmproto.Header{Time: tmtime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	// banktypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// setup  block max gas
	if maxBlockGas == 0 {
		maxBlockGas = MaxBlockGas
	}

	return keeper.NewKeeper(encCfg.Codec, key, &types.ErrorDenomResolver{}, authority), ctx
}
