package testutil

import (
	"testing"

	"github.com/golang/mock/gomock"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtime "github.com/cometbft/cometbft/types/time"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/coredaos/keeper"
	"github.com/atomone-hub/atomone/x/coredaos/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

type Mocks struct {
	GovKeeper     *MockGovKeeper
	StakingKeeper *MockStakingKeeper
}

func SetupMsgServer(t *testing.T) (types.MsgServer, *keeper.Keeper, Mocks, sdk.Context) {
	t.Helper()
	k, m, ctx := SetupCoredaosKeeper(t)
	return keeper.NewMsgServer(k), k, m, ctx
}

func SetupCoredaosKeeper(t *testing.T) (
	*keeper.Keeper,
	Mocks,
	sdk.Context,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := Mocks{
		GovKeeper:     NewMockGovKeeper(ctrl),
		StakingKeeper: NewMockStakingKeeper(ctrl),
	}

	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(tmproto.Header{Time: tmtime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	return keeper.NewKeeper(encCfg.Codec, storeService, authority, m.GovKeeper, m.StakingKeeper), m, ctx
}
