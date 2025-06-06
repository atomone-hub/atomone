package fuzz_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/cometbft/cometbft/libs/log"

	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/x/feemarket/types"
)

// TestLearningRate ensures that the learning rate is always
// constant for the default EIP-1559 implementation.
func TestLearningRate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		state := types.DefaultState()
		params, maxBlockGas := CreateRandomParams(t)

		// Randomly generate alpha and beta.
		prevLearningRate := state.LearningRate

		// Randomly generate the block gas.
		blockGas := rapid.Uint64Range(0, maxBlockGas).Draw(t, "gas")

		// Update the fee market.
		if err := state.Update(blockGas, maxBlockGas); err != nil {
			t.Fatalf("block update errors: %v", err)
		}

		// Update the learning rate.
		lr := state.UpdateLearningRate(params, maxBlockGas)
		require.Equal(t, types.DefaultMinLearningRate, lr)
		require.Equal(t, prevLearningRate, state.LearningRate)
	})
}

// TestGasPrice ensures that the gas price moves in the correct
// direction for the default EIP-1559 implementation.
func TestGasPrice(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		state := types.DefaultState()
		params, maxBlockGas := CreateRandomParams(t)

		// Update the current base fee to be 10% higher than the minimum base fee.
		prevBaseGasPrice := state.BaseGasPrice.Mul(math.LegacyNewDec(11)).Quo(math.LegacyNewDec(10))
		state.BaseGasPrice = prevBaseGasPrice

		// Randomly generate the block gas.
		blockGas := rapid.Uint64Range(0, maxBlockGas).Draw(t, "gas")

		// Update the fee market.
		if err := state.Update(blockGas, maxBlockGas); err != nil {
			t.Fatalf("block update errors: %v", err)
		}

		// Update the learning rate.
		state.UpdateLearningRate(params, maxBlockGas)
		// Update the base fee.
		state.UpdateBaseGasPrice(log.NewNopLogger(), params, maxBlockGas)

		// Ensure that the minimum base fee is always less than the base fee.
		require.True(t, params.MinBaseGasPrice.LTE(state.BaseGasPrice))

		switch {
		case blockGas > types.GetTargetBlockGas(maxBlockGas, params):
			require.True(t, state.BaseGasPrice.GTE(prevBaseGasPrice))
		case blockGas < types.GetTargetBlockGas(maxBlockGas, params):
			require.True(t, state.BaseGasPrice.LTE(prevBaseGasPrice))
		default:
			require.Equal(t, state.BaseGasPrice, prevBaseGasPrice)
		}
	})
}

// CreateRandomParams returns a random set of parameters for the default
// EIP-1559 fee market implementation.
func CreateRandomParams(t *rapid.T) (types.Params, uint64) {
	a := rapid.Uint64Range(1, 1000).Draw(t, "alpha")
	alpha := math.LegacyNewDec(int64(a)).Quo(math.LegacyNewDec(1000))

	b := rapid.Uint64Range(50, 99).Draw(t, "beta")
	beta := math.LegacyNewDec(int64(b)).Quo(math.LegacyNewDec(100))

	g := rapid.Uint64Range(10, 50).Draw(t, "gamma")
	gamma := math.LegacyNewDec(int64(g)).Quo(math.LegacyNewDec(100))

	maxBlockGas := rapid.Uint64Range(2, 30_000_000).Draw(t, "max_block_gas")

	params := types.DefaultParams()
	params.Alpha = alpha
	params.Beta = beta
	params.Gamma = gamma

	return params, maxBlockGas
}
