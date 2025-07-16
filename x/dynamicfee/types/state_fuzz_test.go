package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/atomone-hub/atomone/x/dynamicfee/testutil"
	"github.com/atomone-hub/atomone/x/dynamicfee/types"
)

func FuzzDefaultDynamicfee(f *testing.F) {
	testCases := []uint64{
		0,
		1_000,
		10_000,
		100_000,
		1_000_000,
		10_000_000,
		100_000_000,
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	defaultLR := math.LegacyMustNewDecFromStr("0.125")

	// Default dynamicfee.
	f.Fuzz(func(t *testing.T, blockGasUsed uint64) {
		state := types.DefaultState()
		params := types.DefaultParams()

		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("100")
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("200")
		err := state.Update(blockGasUsed, testutil.MaxBlockGas)

		if blockGasUsed > testutil.MaxBlockGas {
			require.ErrorIs(t, err, types.ErrMaxGasExceeded)
			return
		}

		require.NoError(t, err)
		require.Equal(t, blockGasUsed, state.Window[state.Index])

		// Ensure the learning rate is always the default learning rate.
		lr := state.UpdateLearningRate(
			params,
			testutil.MaxBlockGas,
		)
		require.Equal(t, defaultLR, lr)

		oldFee := state.BaseGasPrice
		newFee := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		if blockGasUsed > types.GetTargetBlockGas(testutil.MaxBlockGas, params) {
			require.True(t, newFee.GT(oldFee))
		} else {
			require.True(t, newFee.LT(oldFee))
		}
	})
}

func FuzzAIMDDynamicfee(f *testing.F) {
	testCases := []uint64{
		0,
		1_000,
		10_000,
		100_000,
		1_000_000,
		10_000_000,
		100_000_000,
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	// Dynamic fee pricing with adjustable learning rate.
	f.Fuzz(func(t *testing.T, blockGasUsed uint64) {
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("100")
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("200")
		state.Window = make([]uint64, 1)
		err := state.Update(blockGasUsed, testutil.MaxBlockGas)

		if blockGasUsed > testutil.MaxBlockGas {
			require.ErrorIs(t, err, types.ErrMaxGasExceeded)
			return
		}

		require.NoError(t, err)
		require.Equal(t, blockGasUsed, state.Window[state.Index])

		oldFee := state.BaseGasPrice
		newFee := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		if blockGasUsed > types.GetTargetBlockGas(testutil.MaxBlockGas, params) {
			require.True(t, newFee.GT(oldFee))
		} else {
			require.True(t, newFee.LT(oldFee))
		}
	})
}
