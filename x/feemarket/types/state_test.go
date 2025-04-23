package types_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/libs/log"

	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

var OneHundred = math.LegacyMustNewDecFromStr("100")

func TestState_Update(t *testing.T) {
	t.Run("can add to window", func(t *testing.T) {
		state := types.DefaultState()

		err := state.Update(100, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(100), state.Window[0])
	})

	t.Run("can add several txs to window", func(t *testing.T) {
		state := types.DefaultState()

		err := state.Update(100, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(100), state.Window[0])

		err = state.Update(200, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(300), state.Window[0])
	})

	t.Run("errors when it exceeds max block utilization", func(t *testing.T) {
		state := types.DefaultState()

		err := state.Update(testutil.MaxBlockGas+1, testutil.MaxBlockGas)
		require.Error(t, err)
	})

	t.Run("can update with several blocks in default eip-1559", func(t *testing.T) {
		state := types.DefaultState()

		err := state.Update(100, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(100), state.Window[0])

		state.IncrementHeight()

		err = state.Update(200, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(200), state.Window[0])

		err = state.Update(300, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(500), state.Window[0])
	})

	t.Run("can update with several blocks in default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		err := state.Update(100, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(100), state.Window[0])

		state.IncrementHeight()

		err = state.Update(200, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(200), state.Window[1])

		state.IncrementHeight()

		err = state.Update(300, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(300), state.Window[2])

		state.IncrementHeight()

		err = state.Update(400, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(400), state.Window[3])
	})

	t.Run("correctly wraps around with aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		state.Window = make([]uint64, 3)

		err := state.Update(100, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(100), state.Window[0])

		state.IncrementHeight()

		err = state.Update(200, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(200), state.Window[1])

		state.IncrementHeight()

		err = state.Update(300, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(300), state.Window[2])

		state.IncrementHeight()
		require.Equal(t, uint64(0), state.Window[0])

		err = state.Update(400, testutil.MaxBlockGas)
		require.NoError(t, err)
		require.Equal(t, uint64(400), state.Window[0])
		require.Equal(t, uint64(200), state.Window[1])
		require.Equal(t, uint64(300), state.Window[2])
	})
}

func TestState_UpdateBaseGasPrice(t *testing.T) {
	t.Run("empty block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("1000")
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("125")

		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := math.LegacyMustNewDecFromStr("875")
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("target block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("1000")
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("125")
		state.Window[0] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)

		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := math.LegacyMustNewDecFromStr("1000")
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("full block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("1000")
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("125")
		state.Window[0] = testutil.MaxBlockGas

		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := math.LegacyMustNewDecFromStr("1125")
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("empty block with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("1000")
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("125")
		state.LearningRate = math.LegacyMustNewDecFromStr("0.125")
		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := math.LegacyMustNewDecFromStr("850")
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("target block with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("1000")
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("125")
		state.LearningRate = math.LegacyMustNewDecFromStr("0.125")
		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)
		}
		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := math.LegacyMustNewDecFromStr("1000")
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("full blocks with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()
		state.BaseGasPrice = math.LegacyMustNewDecFromStr("1000")
		params.MinBaseGasPrice = math.LegacyMustNewDecFromStr("125")
		state.LearningRate = math.LegacyMustNewDecFromStr("0.125")
		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = testutil.MaxBlockGas
		}
		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := math.LegacyMustNewDecFromStr("1150")
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("never goes below min base gas price with default eip1599", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()

		// Empty block
		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := params.MinBaseGasPrice
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("never goes below min base gas price with default aimd eip1599", func(t *testing.T) {
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()

		// Empty blocks
		newBaseGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		require.Equal(t, newBaseGasPrice, state.BaseGasPrice)
		expectedBaseGasPrice := params.MinBaseGasPrice
		require.Equal(t, expectedBaseGasPrice, newBaseGasPrice)
	})

	t.Run("half target block size with aimd eip1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		state.Window = make([]uint64, 1)
		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyNewDec(10))
		prevGasPrice := state.BaseGasPrice

		params := types.DefaultAIMDParams()
		params.Window = 1

		// 1/4th of the window is full.
		state.Window[0] = types.GetTargetBlockUtilization(testutil.MaxBlockGas) / 2

		prevLR := state.LearningRate
		lr := state.UpdateLearningRate(params, testutil.MaxBlockGas)
		newGasPrice := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		expectedLR := prevLR.Add(params.Alpha)
		expectedUtilization := math.LegacyMustNewDecFromStr("-0.5")
		expectedLRAdjustment := (expectedLR.Mul(expectedUtilization)).Add(math.LegacyOneDec())

		expectedGasPrice := prevGasPrice.Mul(expectedLRAdjustment)

		require.Equal(t, expectedLR, lr)
		require.Equal(t, expectedGasPrice, newGasPrice)
	})

	t.Run("3/4 max block size with aimd eip1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		state.Window = make([]uint64, 1)
		state.BaseGasPrice = state.BaseGasPrice.Mul(math.LegacyMustNewDecFromStr("10"))
		prevBGS := state.BaseGasPrice

		params := types.DefaultAIMDParams()
		params.Window = 1

		// 1/4th of the window is full.
		state.Window[0] = testutil.MaxBlockGas / 4 * 3

		prevLR := state.LearningRate
		lr := state.UpdateLearningRate(params, testutil.MaxBlockGas)
		bgs := state.UpdateBaseGasPrice(log.NewNopLogger(), params, testutil.MaxBlockGas)

		expectedUtilization := math.LegacyMustNewDecFromStr("0.5")
		expectedLR := prevLR.Add(params.Alpha)
		expectedLRAdjustment := (expectedLR.Mul(expectedUtilization)).Add(math.LegacyOneDec())

		expectedGasPrice := prevBGS.Mul(expectedLRAdjustment)
		require.Equal(t, expectedLR, lr)
		require.Equal(t, expectedGasPrice, bgs)
	})
}

func TestState_UpdateLearningRate(t *testing.T) {
	t.Run("empty block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := math.LegacyMustNewDecFromStr("0.125")
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("target block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()

		state.Window[0] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)

		state.UpdateLearningRate(params, testutil.MaxBlockGas)
		expectedLearningRate := math.LegacyMustNewDecFromStr("0.125")
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("full block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()
		state.Window[0] = testutil.MaxBlockGas

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := math.LegacyMustNewDecFromStr("0.125")
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("between 0 and target with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()
		state.Window[0] = 50000

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := math.LegacyMustNewDecFromStr("0.125")
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("between target and max with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()
		state.Window[0] = types.GetTargetBlockUtilization(testutil.MaxBlockGas) + 50000

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := math.LegacyMustNewDecFromStr("0.125")
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("random value with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()
		params := types.DefaultParams()
		randomValue := rand.Int63n(1000000000)
		state.Window[0] = uint64(randomValue)

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := math.LegacyMustNewDecFromStr("0.125")
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("empty block with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := params.MinLearningRate.Add(params.Alpha)
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("target block with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		defaultLR := math.LegacyMustNewDecFromStr("0.125")
		state.LearningRate = defaultLR
		params := types.DefaultAIMDParams()
		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)
		}

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := defaultLR.Mul(params.Beta)
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("full blocks with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		defaultLR := math.LegacyMustNewDecFromStr("0.125")
		state.LearningRate = defaultLR
		params := types.DefaultAIMDParams()
		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = testutil.MaxBlockGas
		}

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := defaultLR.Add(params.Alpha)
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("varying blocks with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		defaultLR := math.LegacyMustNewDecFromStr("0.125")
		state.LearningRate = defaultLR
		params := types.DefaultAIMDParams()
		for i := 0; i < len(state.Window); i++ {
			if i%2 == 0 {
				state.Window[i] = testutil.MaxBlockGas
			} else {
				state.Window[i] = 0
			}
		}

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := defaultLR.Mul(params.Beta)
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})

	t.Run("exceeds threshold with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		defaultLR := math.LegacyMustNewDecFromStr("0.125")
		state.LearningRate = defaultLR
		params := types.DefaultAIMDParams()
		for i := 0; i < len(state.Window); i++ {
			if i%2 == 0 {
				state.Window[i] = testutil.MaxBlockGas
			} else {
				state.Window[i] = types.GetTargetBlockUtilization(testutil.MaxBlockGas) + 1
			}
		}

		state.UpdateLearningRate(params, testutil.MaxBlockGas)

		expectedLearningRate := defaultLR.Add(params.Alpha)
		require.Equal(t, expectedLearningRate, state.LearningRate)
	})
}

func TestState_GetNetUtilization(t *testing.T) {
	t.Run("empty block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()

		netUtilization := state.GetNetUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.NewInt(0).Sub(math.NewIntFromUint64(types.GetTargetBlockUtilization(testutil.MaxBlockGas)))
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("target block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()

		state.Window[0] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)

		netUtilization := state.GetNetUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.NewInt(0)
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("full block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()

		state.Window[0] = testutil.MaxBlockGas

		netUtilization := state.GetNetUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.NewIntFromUint64(testutil.MaxBlockGas - types.GetTargetBlockUtilization(testutil.MaxBlockGas))
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("empty block with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		netUtilization := state.GetNetUtilization(testutil.MaxBlockGas)

		multiple := math.NewIntFromUint64(uint64(len(state.Window)))
		expectedUtilization := math.NewInt(0).Sub(math.NewIntFromUint64(types.GetTargetBlockUtilization(testutil.MaxBlockGas))).Mul(multiple)
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("full blocks with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = testutil.MaxBlockGas
		}

		netUtilization := state.GetNetUtilization(testutil.MaxBlockGas)

		multiple := math.NewIntFromUint64(uint64(len(state.Window)))
		expectedUtilization := math.NewIntFromUint64(testutil.MaxBlockGas).Sub(math.NewIntFromUint64(types.GetTargetBlockUtilization(testutil.MaxBlockGas))).Mul(multiple)
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("varying blocks with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		for i := 0; i < len(state.Window); i++ {
			if i%2 == 0 {
				state.Window[i] = testutil.MaxBlockGas
			} else {
				state.Window[i] = 0
			}
		}

		netUtilization := state.GetNetUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.ZeroInt()
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("exceeds target rate with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()
		params := types.DefaultAIMDParams()

		for i := 0; i < len(state.Window); i++ {
			if i%2 == 0 {
				state.Window[i] = testutil.MaxBlockGas
			} else {
				state.Window[i] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)
			}
		}

		netUtilization := state.GetNetUtilization(testutil.MaxBlockGas)
		first := math.NewIntFromUint64(testutil.MaxBlockGas).Mul(math.NewIntFromUint64(params.Window / 2))
		second := math.NewIntFromUint64(types.GetTargetBlockUtilization(testutil.MaxBlockGas)).Mul(math.NewIntFromUint64(params.Window / 2))
		expectedUtilization := first.Add(second).Sub(math.NewIntFromUint64(types.GetTargetBlockUtilization(testutil.MaxBlockGas)).Mul(math.NewIntFromUint64(params.Window)))
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("state with 4 entries in window with different updates", func(t *testing.T) {
		state := types.DefaultAIMDState()
		state.Window = make([]uint64, 4)

		maxBlockGas := uint64(200)

		state.Window[0] = 100
		state.Window[1] = 200
		state.Window[2] = 0
		state.Window[3] = 50

		netUtilization := state.GetNetUtilization(maxBlockGas)
		expectedUtilization := math.NewIntFromUint64(50).Mul(math.NewInt(-1))
		require.True(t, expectedUtilization.Equal(netUtilization))
	})

	t.Run("state with 4 entries in window with monotonically increasing updates", func(t *testing.T) {
		state := types.DefaultAIMDState()
		state.Window = make([]uint64, 4)

		maxBlockGas := uint64(200)

		state.Window[0] = 0
		state.Window[1] = 25
		state.Window[2] = 50
		state.Window[3] = 75

		netUtilization := state.GetNetUtilization(maxBlockGas)
		expectedUtilization := math.NewIntFromUint64(250).Mul(math.NewInt(-1))
		require.True(t, expectedUtilization.Equal(netUtilization))
	})
}

func TestState_GetAverageUtilization(t *testing.T) {
	t.Run("empty block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyZeroDec()
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("target block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()

		state.Window[0] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("0.5")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("full block with default eip-1559", func(t *testing.T) {
		state := types.DefaultState()

		state.Window[0] = testutil.MaxBlockGas

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("1.0")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("empty block with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyZeroDec()
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("target block with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)
		}

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("0.5")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("full blocks with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		for i := 0; i < len(state.Window); i++ {
			state.Window[i] = testutil.MaxBlockGas
		}

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("1.0")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("varying blocks with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		for i := 0; i < len(state.Window); i++ {
			if i%2 == 0 {
				state.Window[i] = testutil.MaxBlockGas
			} else {
				state.Window[i] = 0
			}
		}

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("0.5")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("exceeds target rate with default aimd eip-1559", func(t *testing.T) {
		state := types.DefaultAIMDState()

		for i := 0; i < len(state.Window); i++ {
			if i%2 == 0 {
				state.Window[i] = testutil.MaxBlockGas
			} else {
				state.Window[i] = types.GetTargetBlockUtilization(testutil.MaxBlockGas)
			}
		}

		avgUtilization := state.GetAverageUtilization(testutil.MaxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("0.75")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("state with 4 entries in window with different updates", func(t *testing.T) {
		state := types.DefaultAIMDState()
		state.Window = make([]uint64, 4)

		maxBlockGas := uint64(200)

		state.Window[0] = 100
		state.Window[1] = 200
		state.Window[2] = 0
		state.Window[3] = 50

		avgUtilization := state.GetAverageUtilization(maxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("0.4375")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})

	t.Run("state with 4 entries in window with monotonically increasing updates", func(t *testing.T) {
		state := types.DefaultAIMDState()
		state.Window = make([]uint64, 4)

		params := types.DefaultAIMDParams()
		params.Window = 4
		maxBlockGas := uint64(200)

		state.Window[0] = 0
		state.Window[1] = 25
		state.Window[2] = 50
		state.Window[3] = 75

		avgUtilization := state.GetAverageUtilization(maxBlockGas)
		expectedUtilization := math.LegacyMustNewDecFromStr("0.1875")
		require.True(t, expectedUtilization.Equal(avgUtilization))
	})
}

func TestState_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name      string
		state     types.State
		expectErr bool
	}{
		{
			name:      "default base EIP-1559 state",
			state:     types.DefaultState(),
			expectErr: false,
		},
		{
			name:      "default AIMD EIP-1559 state",
			state:     types.DefaultAIMDState(),
			expectErr: false,
		},
		{
			name:      "invalid window",
			state:     types.State{},
			expectErr: true,
		},
		{
			name: "invalid negative base gas price",
			state: types.State{
				Window:       make([]uint64, 1),
				BaseGasPrice: math.LegacyMustNewDecFromStr("-1"),
			},
			expectErr: true,
		},
		{
			name: "invalid learning rate",
			state: types.State{
				Window:       make([]uint64, 1),
				BaseGasPrice: math.LegacyMustNewDecFromStr("1"),
				LearningRate: math.LegacyMustNewDecFromStr("-1.0"),
			},
			expectErr: true,
		},
		{
			name: "valid other state",
			state: types.State{
				Window:       make([]uint64, 1),
				BaseGasPrice: math.LegacyMustNewDecFromStr("1"),
				LearningRate: math.LegacyMustNewDecFromStr("0.5"),
			},
			expectErr: false,
		},
		{
			name: "invalid zero base gas price",
			state: types.State{
				Window:       make([]uint64, 1),
				BaseGasPrice: math.LegacyZeroDec(),
				LearningRate: math.LegacyMustNewDecFromStr("0.5"),
			},
			expectErr: true,
		},
		{
			name: "invalid zero learning rate",
			state: types.State{
				Window:       make([]uint64, 1),
				BaseGasPrice: math.LegacyMustNewDecFromStr("1"),
				LearningRate: math.LegacyZeroDec(),
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.state.ValidateBasic()
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
