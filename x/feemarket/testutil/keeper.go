package testutil

import (
	"testing"

	"github.com/golang/mock/gomock"

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

type Mocks struct {
	AccountKeeper  *MockAccountKeeper
	BankKeeper     *MockBankKeeper
	FeeGrantKeeper *MockFeeGrantKeeper
}

func SetupMsgServer(t *testing.T) (types.MsgServer, *keeper.Keeper, Mocks, sdk.Context) {
	t.Helper()
	k, m, ctx := SetupFeemarketKeeper(t)
	return keeper.NewMsgServer(k), k, m, ctx
}

func SetupQueryServer(t *testing.T) (types.QueryServer, *keeper.Keeper, Mocks, sdk.Context) {
	t.Helper()
	k, m, ctx := SetupFeemarketKeeper(t)
	return keeper.NewQueryServer(*k), k, m, ctx
}

func SetupFeemarketKeeper(t *testing.T) (
	*keeper.Keeper,
	Mocks,
	sdk.Context,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := Mocks{
		AccountKeeper:  NewMockAccountKeeper(ctrl),
		BankKeeper:     NewMockBankKeeper(ctrl),
		FeeGrantKeeper: NewMockFeeGrantKeeper(ctrl),
	}

	key := sdk.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, sdk.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(tmproto.Header{Time: tmtime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	// banktypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	k := keeper.NewKeeper(encCfg.Codec, key, &types.ErrorDenomResolver{}, authority)
	return k, m, ctx
}
