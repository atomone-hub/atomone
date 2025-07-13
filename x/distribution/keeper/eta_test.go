package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/x/distribution"
	"github.com/atomone-hub/atomone/x/distribution/keeper"
	distrtestutil "github.com/atomone-hub/atomone/x/distribution/testutil"
	"github.com/atomone-hub/atomone/x/distribution/types"
)

func createValidators(powers ...int64) ([]stakingtypes.Validator, error) {
	vals := make([]stakingtypes.Validator, len(powers))
	for i, p := range powers {
		vals[i] = stakingtypes.Validator{
			OperatorAddress: sdk.ValAddress([]byte{byte(i)}).String(),
			Tokens:          math.NewInt(p),
			Status:          stakingtypes.Bonded,
		}
	}
	return vals, nil
}

func TestAdjustEta_NoInterval(t *testing.T) {
	ctrl := gomock.NewController(t)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(distribution.AppModuleBasic{})
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now(), Height: 119_999})

	bankKeeper := distrtestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := distrtestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := distrtestutil.NewMockAccountKeeper(ctrl)

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(distrAcc.GetAddress())

	distrKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		"fee_collector",
		authtypes.NewModuleAddress("gov").String(),
	)

	initEta := math.LegacyNewDecWithPrec(3, 2)
	params := types.DefaultParams()
	params.NakamotoBonusCoefficient = initEta
	require.NoError(t, distrKeeper.Params.Set(ctx, params))

	err := distrKeeper.AdjustEta(ctx)
	require.NoError(t, err)
	gotParams, _ := distrKeeper.Params.Get(ctx)
	require.Equal(t, initEta, gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_NotEnoughValidators(t *testing.T) {
	ctrl := gomock.NewController(t)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(distribution.AppModuleBasic{})
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now(), Height: 120_000})

	bankKeeper := distrtestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := distrtestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := distrtestutil.NewMockAccountKeeper(ctrl)

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(distrAcc.GetAddress())
	stakingKeeper.EXPECT().GetBondedValidatorsByPower(ctx).Return(createValidators(10, 10))

	distrKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		"fee_collector",
		authtypes.NewModuleAddress("gov").String(),
	)

	initEta := math.LegacyNewDecWithPrec(3, 2)
	params := types.DefaultParams()
	params.NakamotoBonusCoefficient = initEta
	require.NoError(t, distrKeeper.Params.Set(ctx, params))

	err := distrKeeper.AdjustEta(ctx)
	require.NoError(t, err)
	gotParams, _ := distrKeeper.Params.Get(ctx)
	require.Equal(t, initEta, gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_Increase(t *testing.T) {
	ctrl := gomock.NewController(t)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(distribution.AppModuleBasic{})
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now(), Height: 120_000})

	bankKeeper := distrtestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := distrtestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := distrtestutil.NewMockAccountKeeper(ctrl)

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(distrAcc.GetAddress())
	// highAvg = 100, lowAvg = 10, ratio = 10 >= 3, should increase
	stakingKeeper.EXPECT().GetBondedValidatorsByPower(ctx).Return(createValidators(100, 100, 10))

	distrKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		"fee_collector",
		authtypes.NewModuleAddress("gov").String(),
	)

	initEta := math.LegacyNewDecWithPrec(3, 2)
	params := types.DefaultParams()
	params.NakamotoBonusCoefficient = initEta
	require.NoError(t, distrKeeper.Params.Set(ctx, params))

	err := distrKeeper.AdjustEta(ctx)
	require.NoError(t, err)
	gotParams, _ := distrKeeper.Params.Get(ctx)
	require.Equal(t, initEta.Add(math.LegacyNewDecWithPrec(3, 2)), gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_Decrease(t *testing.T) {
	ctrl := gomock.NewController(t)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(distribution.AppModuleBasic{})
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now(), Height: 120_000})

	bankKeeper := distrtestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := distrtestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := distrtestutil.NewMockAccountKeeper(ctrl)

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(distrAcc.GetAddress())
	// highAvg = 20, lowAvg = 10, ratio = 2 < 3, should decrease
	stakingKeeper.EXPECT().GetBondedValidatorsByPower(ctx).Return(createValidators(20, 20, 10))

	distrKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		"fee_collector",
		authtypes.NewModuleAddress("gov").String(),
	)

	initEta := math.LegacyNewDecWithPrec(3, 2)
	params := types.DefaultParams()
	params.NakamotoBonusCoefficient = initEta
	require.NoError(t, distrKeeper.Params.Set(ctx, params))

	err := distrKeeper.AdjustEta(ctx)
	require.NoError(t, err)
	gotParams, _ := distrKeeper.Params.Get(ctx)
	require.Equal(t, math.LegacyZeroDec(), gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_ClampZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(distribution.AppModuleBasic{})
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now(), Height: 120_000})

	bankKeeper := distrtestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := distrtestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := distrtestutil.NewMockAccountKeeper(ctrl)

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(distrAcc.GetAddress())
	// highAvg = 20, lowAvg = 10, ratio = 2 < 3, should decrease, and clamp at 0
	stakingKeeper.EXPECT().GetBondedValidatorsByPower(ctx).Return(createValidators(20, 20, 10))

	distrKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		"fee_collector",
		authtypes.NewModuleAddress("gov").String(),
	)

	initEta := math.LegacyZeroDec()
	params := types.DefaultParams()
	params.NakamotoBonusCoefficient = initEta
	require.NoError(t, distrKeeper.Params.Set(ctx, params))

	err := distrKeeper.AdjustEta(ctx)
	require.NoError(t, err)
	gotParams, _ := distrKeeper.Params.Get(ctx)
	require.True(t, gotParams.NakamotoBonusCoefficient.GTE(math.LegacyZeroDec()))
}

func TestAdjustEta_ClampOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(distribution.AppModuleBasic{})
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now(), Height: 120_000})

	bankKeeper := distrtestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := distrtestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := distrtestutil.NewMockAccountKeeper(ctrl)

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(distrAcc.GetAddress())
	// highAvg = 100, lowAvg = 10, ratio = 10 >= 3, should increase
	stakingKeeper.EXPECT().GetBondedValidatorsByPower(ctx).Return(createValidators(100, 100, 10))

	distrKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		"fee_collector",
		authtypes.NewModuleAddress("gov").String(),
	)

	initEta := math.LegacyOneDec()
	params := types.DefaultParams()
	params.NakamotoBonusCoefficient = initEta
	require.NoError(t, distrKeeper.Params.Set(ctx, params))

	err := distrKeeper.AdjustEta(ctx)
	require.NoError(t, err)
	gotParams, _ := distrKeeper.Params.Get(ctx)
	require.True(t, gotParams.NakamotoBonusCoefficient.LTE(math.LegacyOneDec()))
}
