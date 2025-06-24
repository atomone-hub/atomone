package testutil

import (
	"testing"

	"github.com/golang/mock/gomock"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtime "github.com/cometbft/cometbft/types/time"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/feemarket/keeper"
	"github.com/atomone-hub/atomone/x/feemarket/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

type Mocks struct {
	ConsensusParamsKeeper *MockConsensusParamsKeeper
}

func SetupMsgServer(t *testing.T, maxBlockGas uint64) (types.MsgServer, *keeper.Keeper, Mocks, sdk.Context) {
	t.Helper()
	k, m, ctx := SetupKeeper(t, maxBlockGas)
	return keeper.NewMsgServer(k), k, m, ctx
}

func SetupQueryServer(t *testing.T, maxBlockGas uint64) (types.QueryServer, *keeper.Keeper, Mocks, sdk.Context) {
	t.Helper()
	k, m, ctx := SetupKeeper(t, maxBlockGas)
	return keeper.NewQueryServer(*k), k, m, ctx
}

const MaxBlockGas = 30_000_000

func SetupKeeper(t *testing.T, maxBlockGas uint64) (*keeper.Keeper, Mocks, sdk.Context) {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := Mocks{
		ConsensusParamsKeeper: NewMockConsensusParamsKeeper(ctrl),
	}

	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(tmproto.Header{Time: tmtime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	// banktypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// setup  block max gas
	if maxBlockGas == 0 {
		maxBlockGas = MaxBlockGas
	}
	m.ConsensusParamsKeeper.EXPECT().Get(ctx).
		Return(tmproto.ConsensusParams{
			Block: &tmproto.BlockParams{MaxGas: int64(maxBlockGas)},
		}, nil).MaxTimes(1)

	return keeper.NewKeeper(encCfg.Codec, key, &types.ErrorDenomResolver{}, m.ConsensusParamsKeeper, authority), m, ctx
}
