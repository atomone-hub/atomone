package module_test

import (
	"testing"

	"github.com/atomone-hub/atomone/baseapp"
	"github.com/atomone-hub/atomone/testutil"
	simtestutil "github.com/atomone-hub/atomone/testutil/sims"
	sdk "github.com/atomone-hub/atomone/types"
	moduletestutil "github.com/atomone-hub/atomone/types/module/testutil"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	"github.com/atomone-hub/atomone/x/feegrant"
	"github.com/atomone-hub/atomone/x/feegrant/keeper"
	"github.com/atomone-hub/atomone/x/feegrant/module"
	feegranttestutil "github.com/atomone-hub/atomone/x/feegrant/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestFeegrantPruning(t *testing.T) {
	key := sdk.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, sdk.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModuleBasic{})

	addrs := simtestutil.CreateIncrementalAccounts(4)
	granter1 := addrs[0]
	granter2 := addrs[1]
	granter3 := addrs[2]
	grantee := addrs[3]
	spendLimit := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))
	now := testCtx.Ctx.BlockTime()
	oneDay := now.AddDate(0, 0, 1)

	ctrl := gomock.NewController(t)
	accountKeeper := feegranttestutil.NewMockAccountKeeper(ctrl)
	accountKeeper.EXPECT().GetAccount(gomock.Any(), grantee).Return(authtypes.NewBaseAccountWithAddress(grantee)).AnyTimes()
	accountKeeper.EXPECT().GetAccount(gomock.Any(), granter1).Return(authtypes.NewBaseAccountWithAddress(granter1)).AnyTimes()
	accountKeeper.EXPECT().GetAccount(gomock.Any(), granter2).Return(authtypes.NewBaseAccountWithAddress(granter2)).AnyTimes()
	accountKeeper.EXPECT().GetAccount(gomock.Any(), granter3).Return(authtypes.NewBaseAccountWithAddress(granter3)).AnyTimes()

	feegrantKeeper := keeper.NewKeeper(encCfg.Codec, key, accountKeeper)

	feegrantKeeper.GrantAllowance(
		testCtx.Ctx,
		granter1,
		grantee,
		&feegrant.BasicAllowance{
			Expiration: &now,
		},
	)
	feegrantKeeper.GrantAllowance(
		testCtx.Ctx,
		granter2,
		grantee,
		&feegrant.BasicAllowance{
			SpendLimit: spendLimit,
		},
	)
	feegrantKeeper.GrantAllowance(
		testCtx.Ctx,
		granter3,
		grantee,
		&feegrant.BasicAllowance{
			Expiration: &oneDay,
		},
	)

	queryHelper := baseapp.NewQueryServerTestHelper(testCtx.Ctx, encCfg.InterfaceRegistry)
	feegrant.RegisterQueryServer(queryHelper, feegrantKeeper)
	queryClient := feegrant.NewQueryClient(queryHelper)

	module.EndBlocker(testCtx.Ctx, feegrantKeeper)

	res, err := queryClient.Allowances(testCtx.Ctx.Context(), &feegrant.QueryAllowancesRequest{
		Grantee: grantee.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Allowances, 3)

	testCtx.Ctx = testCtx.Ctx.WithBlockTime(now.AddDate(0, 0, 2))
	module.EndBlocker(testCtx.Ctx, feegrantKeeper)

	res, err = queryClient.Allowances(testCtx.Ctx.Context(), &feegrant.QueryAllowancesRequest{
		Grantee: grantee.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Allowances, 1)
}