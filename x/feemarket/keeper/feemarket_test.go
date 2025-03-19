package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

func TestUpdateFeeMarket(t *testing.T) {
	t.Run("empty block with default eip1559 with min base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		params := types.DefaultParams()
		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, params.MinBaseGasPrice)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("empty block with default eip1559 with preset base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(2))
		params := types.DefaultParams()
		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})
		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to decrease by 1/8th.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)

		factor := math.LegacyMustNewDecFromStr("0.875")
		expectedFee := state.BaseGasPrice.Mul(factor)
		require.Equal(fee, expectedFee)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("empty block with default eip1559 with preset base fee < 1", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		// this value should be ignored -> 0.5 should be used instead
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("0.25")

		// change MinBaseGasPrice value < 1
		params := types.DefaultParams()
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("0.5")
		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)

		require.Equal(fee, math.LegacyMustNewDecFromStr("0.5"))

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("empty block default eip1559 with preset base fee that should default to min", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		// Set the base fee to just below the expected threshold decrease of 1/8th. This means it
		// should default to the minimum base fee.
		state := types.DefaultState()
		factor := math.LegacyMustNewDecFromStr("0.125")
		change := state.BaseGasPrice.Mul(factor)
		state.BaseGasPrice = types.DefaultMinBaseGasPrice.Sub(change)

		params := types.DefaultParams()
		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to decrease by 1/8th.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, params.MinBaseGasPrice)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("target block with default eip1559 at min base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		params := types.DefaultParams()

		// Reaching the target block size means that we expect this to not
		// increase.
		err := state.Update(params.TargetBlockUtilization(), params)
		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to remain the same.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, params.MinBaseGasPrice)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("target block with default eip1559 at preset base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		params := types.DefaultParams()

		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(2))
		// Reaching the target block size means that we expect this to not
		// increase.
		err := state.Update(params.TargetBlockUtilization(), params)
		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to remain the same.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(state.BaseGasPrice, fee)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("max block with default eip1559 at min base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		params := types.DefaultParams()

		// Reaching the target block size means that we expect this to not
		// increase.
		err := state.Update(params.MaxBlockUtilization, params)
		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to increase by 1/8th.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)

		factor := math.LegacyMustNewDecFromStr("1.125")
		expectedFee := state.BaseGasPrice.Mul(factor)
		require.Equal(fee, expectedFee)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("max block with default eip1559 at preset base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		params := types.DefaultParams()

		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(2))
		// Reaching the target block size means that we expect this to not
		// increase.
		err := state.Update(params.MaxBlockUtilization, params)
		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to increase by 1/8th.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)

		factor := math.LegacyMustNewDecFromStr("1.125")
		expectedFee := state.BaseGasPrice.Mul(factor)
		require.Equal(fee, expectedFee)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("in-between min and target block with default eip1559 at min base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		params := types.DefaultParams()
		params.MaxBlockUtilization = 100

		err := state.Update(25, params)
		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to remain the same since it is at min base fee.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, params.MinBaseGasPrice)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("in-between min and target block with default eip1559 at preset base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(2))

		params := types.DefaultParams()
		params.MaxBlockUtilization = 100
		err := state.Update(25, params)

		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to decrease by 1/8th * 1/2.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)

		factor := math.LegacyMustNewDecFromStr("0.9375")
		expectedFee := state.BaseGasPrice.Mul(factor)
		require.Equal(fee, expectedFee)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("in-between target and max block with default eip1559 at min base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		params := types.DefaultParams()
		params.MaxBlockUtilization = 100

		err := state.Update(75, params)
		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to increase by 1/8th * 1/2.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)

		factor := math.LegacyMustNewDecFromStr("1.0625")
		expectedFee := state.BaseGasPrice.Mul(factor)
		require.Equal(fee, expectedFee)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("in-between target and max block with default eip1559 at preset base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultState()
		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(2))
		params := types.DefaultParams()
		params.MaxBlockUtilization = 100

		err := state.Update(75, params)
		require.NoError(err)

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to increase by 1/8th * 1/2.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)

		factor := math.LegacyMustNewDecFromStr("1.0625")
		expectedFee := state.BaseGasPrice.Mul(factor)
		require.Equal(fee, expectedFee)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(math.LegacyMustNewDecFromStr("0.125"), lr)
	})

	t.Run("empty blocks with aimd eip1559 with min base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()
		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, params.MinBaseGasPrice)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		expectedLR := state.LearningRate.Add(params.Alpha)
		require.Equal(expectedLR, lr)
	})

	t.Run("empty blocks with aimd eip1559 with preset base fee", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultAIMDState()
		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(2))
		params := types.DefaultAIMDParams()
		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to decrease by 1/8th and the learning rate to
		// increase by alpha.
		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		expectedLR := state.LearningRate.Add(params.Alpha)
		require.Equal(expectedLR, lr)

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		factor := math.LegacyOneDec().Add(math.LegacyMustNewDecFromStr("-1.0").Mul(lr))
		expectedFee := state.BaseGasPrice.Mul(factor)
		require.Equal(fee, expectedFee)
	})

	t.Run("empty blocks aimd eip1559 with preset base fee that should default to min", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		params := types.DefaultAIMDParams()

		state := types.DefaultAIMDState()
		lr := math.LegacyMustNewDecFromStr("0.125")
		increase := state.BaseGasPrice.Mul(lr).TruncateInt()

		state.BaseGasPrice = types.DefaultMinBaseGasPrice.Add(math.LegacyNewDecFromInt(increase)).Sub(math.LegacyNewDec(1))
		state.LearningRate = lr

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		expectedLR := state.LearningRate.Add(params.Alpha)
		require.Equal(expectedLR, lr)

		// We expect the base fee to decrease by 1/8th and the learning rate to
		// increase by alpha.
		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, params.MinBaseGasPrice)
	})

	t.Run("target block with aimd eip1559 at min base fee + LR", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()

		// Reaching the target block size means that we expect this to not
		// increase.
		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = params.TargetBlockUtilization()
		}

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to remain the same and the learning rate to
		// remain at minimum.
		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(params.MinLearningRate, lr)

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(state.BaseGasPrice, fee)
	})

	t.Run("target block with aimd eip1559 at preset base fee + LR", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		state := types.DefaultAIMDState()
		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(2))
		state.LearningRate = math.LegacyMustNewDecFromStr("0.125")
		params := types.DefaultAIMDParams()

		// Reaching the target block size means that we expect this to not
		// increase.
		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = params.TargetBlockUtilization()
		}

		k.InitGenesis(ctx, types.GenesisState{Params: params, State: state})

		require.NoError(k.UpdateFeeMarket(ctx))

		// We expect the base fee to decrease by 1/8th and the learning rate to
		// decrease by lr * beta.
		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		expectedLR := state.LearningRate.Mul(params.Beta)
		require.Equal(expectedLR, lr)

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(state.BaseGasPrice, fee)
	})
}

func TestGetBaseFee(t *testing.T) {
	t.Run("can retrieve base fee with default eip-1559", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		gs := types.DefaultGenesisState()
		k.InitGenesis(ctx, *gs)

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, gs.State.BaseGasPrice)
	})

	t.Run("can retrieve base fee with aimd eip-1559", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		gs := types.DefaultAIMDGenesisState()
		k.InitGenesis(ctx, *gs)

		fee, err := k.GetBaseGasPrice(ctx)
		require.NoError(err)
		require.Equal(fee, gs.State.BaseGasPrice)
	})
}

func TestGetLearningRate(t *testing.T) {
	t.Run("can retrieve learning rate with default eip-1559", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		gs := types.DefaultGenesisState()
		k.InitGenesis(ctx, *gs)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(lr, gs.State.LearningRate)
	})

	t.Run("can retrieve learning rate with aimd eip-1559", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		gs := types.DefaultAIMDGenesisState()
		k.InitGenesis(ctx, *gs)

		lr, err := k.GetLearningRate(ctx)
		require.NoError(err)
		require.Equal(lr, gs.State.LearningRate)
	})
}

func TestGetMinGasPrices(t *testing.T) {
	t.Run("can retrieve min gas prices with default eip-1559", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		gs := types.DefaultGenesisState()
		k.InitGenesis(ctx, *gs)

		expected := sdk.NewDecCoins(sdk.NewDecCoinFromDec(types.DefaultFeeDenom, gs.State.BaseGasPrice))

		mgp, err := k.GetMinGasPrices(ctx)
		require.NoError(err)
		require.Equal(expected, mgp)
	})

	t.Run("can retrieve min gas prices with aimd eip-1559", func(t *testing.T) {
		require := require.New(t)
		k, _, ctx := testutil.SetupFeemarketKeeper(t)
		gs := types.DefaultAIMDGenesisState()
		k.InitGenesis(ctx, *gs)

		expected := sdk.NewDecCoins(sdk.NewDecCoinFromDec(types.DefaultFeeDenom, gs.State.BaseGasPrice))

		mgp, err := k.GetMinGasPrices(ctx)
		require.NoError(err)
		require.Equal(expected, mgp)
	})
}
