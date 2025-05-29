package fuzz_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/atomone-hub/atomone/x/feemarket/types"
)

// TestAIMDLearningRate ensures that the additive increase
// multiplicative decrease learning rate algorithm correctly
// adjusts the learning rate. In particular, if the block
// gas is greater than theta or less than 1 - theta, then
// the learning rate is increased by the additive increase
// parameter. Otherwise, the learning rate is decreased by
// the multiplicative decrease parameter.
func TestAIMDLearningRate(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		state := types.DefaultAIMDState()
		window := rapid.Int64Range(1, 50).Draw(t, "window")
		state.Window = make([]uint64, window)

		params, maxBlockGas := CreateRandomAIMDParams(t)

		// Randomly generate the block gas
		numBlocks := rapid.Uint64Range(0, 1000).Draw(t, "num_blocks")
		gasGen := rapid.Uint64Range(0, maxBlockGas)

		// Update the fee market.
		for i := uint64(0); i < numBlocks; i++ {
			blockGas := gasGen.Draw(t, "gas")
			prevLearningRate := state.LearningRate

			// Update the fee market.
			if err := state.Update(blockGas, maxBlockGas); err != nil {
				t.Fatalf("block update errors: %v", err)
			}

			// Update the learning rate.
			lr := state.UpdateLearningRate(params, maxBlockGas)
			avgGas := state.GetAverageGas(maxBlockGas)

			// Ensure that the learning rate is always bounded.
			require.True(t, lr.GTE(params.MinLearningRate))
			require.True(t, lr.LTE(params.MaxLearningRate))

			if avgGas.LTE(params.Gamma) || avgGas.GTE(math.LegacyOneDec().Sub(params.Gamma)) {
				require.True(t, lr.GTE(prevLearningRate))
			} else {
				require.True(t, lr.LTE(prevLearningRate))
			}

			// Update the current height.
			state.IncrementHeight()
		}
	})
}

// TestAIMDGasPrice ensures that the additive increase multiplicative
// decrease gas price adjustment algorithm correctly adjusts the base
// fee. In particular, the gas price should function the same as the
// default EIP-1559 base fee adjustment algorithm.
func TestAIMDGasPrice(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		state := types.DefaultAIMDState()
		window := rapid.Int64Range(1, 50).Draw(t, "window")
		state.Window = make([]uint64, window)

		params, maxBlockGas := CreateRandomAIMDParams(t)

		// Randomly generate the block gas.
		numBlocks := rapid.Uint64Range(0, uint64(window)*10).Draw(t, "num_blocks")
		gasGen := rapid.Uint64Range(0, maxBlockGas)

		// Update the fee market.
		for i := uint64(0); i < numBlocks; i++ {
			blockGas := gasGen.Draw(t, "gas")
			prevBaseGasPrice := state.BaseGasPrice

			if err := state.Update(blockGas, maxBlockGas); err != nil {
				t.Fatalf("block update errors: %v", err)
			}

			var total uint64
			for _, gas := range state.Window {
				total += gas
			}

			// Update the learning rate.
			lr := state.UpdateLearningRate(params, maxBlockGas)
			// Update the base gas price.

			var newPrice math.LegacyDec
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						newPrice = params.MinBaseGasPrice
					}
				}()

				// Calculate the new base gasPrice with the learning rate adjustment.
				currentBlockGas := math.LegacyNewDecFromInt(math.NewIntFromUint64(state.Window[state.Index]))
				targetBlockGas := math.LegacyNewDecFromInt(math.NewIntFromUint64(types.GetTargetBlockGas(maxBlockGas)))
				avgGas := (currentBlockGas.Sub(targetBlockGas)).Quo(targetBlockGas)

				// Truncate the learning rate adjustment to an integer.
				//
				// This is equivalent to
				// 1 + (learningRate * (currentBlockGas - targetBlockGas) / targetBlockGas)
				learningRateAdjustment := math.LegacyOneDec().Add(lr.Mul(avgGas))

				// Update the base gasPrice.
				newPrice = prevBaseGasPrice.Mul(learningRateAdjustment)
				// Ensure the base gasPrice is greater than the minimum base gasPrice.
				if newPrice.LT(params.MinBaseGasPrice) {
					newPrice = params.MinBaseGasPrice
				}
			}()

			state.UpdateBaseGasPrice(log.NewNopLogger(), params, maxBlockGas)

			// Ensure that the minimum base fee is always less than the base gas price.
			require.True(t, params.MinBaseGasPrice.LTE(state.BaseGasPrice))

			require.Equal(t, newPrice, state.BaseGasPrice)

			// Update the current height.
			state.IncrementHeight()
		}
	})
}

// CreateRandomAIMDParams returns a random set of parameters for the AIMD
// EIP-1559 fee market implementation.
func CreateRandomAIMDParams(t *rapid.T) (types.Params, uint64) {
	a := rapid.Uint64Range(1, 1000).Draw(t, "alpha")
	alpha := math.LegacyNewDec(int64(a)).Quo(math.LegacyNewDec(1000))

	b := rapid.Uint64Range(50, 99).Draw(t, "beta")
	beta := math.LegacyNewDec(int64(b)).Quo(math.LegacyNewDec(100))

	g := rapid.Uint64Range(10, 50).Draw(t, "gamma")
	gamma := math.LegacyNewDec(int64(g)).Quo(math.LegacyNewDec(100))

	targetBlockGas := rapid.Uint64Range(1, 30_000_000).Draw(t, "target_block_gas")
	maxBlockGas := rapid.Uint64Range(targetBlockGas, targetBlockGas*5).Draw(t, "max_block_gas")

	params := types.DefaultAIMDParams()
	params.Alpha = alpha
	params.Beta = beta
	params.Gamma = gamma

	return params, maxBlockGas
}
