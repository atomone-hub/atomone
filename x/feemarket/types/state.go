package types

import (
	fmt "fmt"

	"cosmossdk.io/log"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
)

// NewState instantiates a new fee market state instance. This is utilized
// to implement both the base EIP-1559 fee market implementation and the
// AIMD EIP-1559 fee market implementation. Note that on init, you initialize
// both the minimum and current base gas price to the same value.
func NewState(
	windowSize uint64,
	baseGasPrice math.LegacyDec,
	learningRate math.LegacyDec,
) State {
	return State{
		Window:       make([]uint64, windowSize),
		BaseGasPrice: baseGasPrice,
		Index:        0,
		LearningRate: learningRate,
	}
}

// Update updates the block gas for the current height with the given
// transaction gas i.e. gas limit.
func (s *State) Update(gas, maxBlockGas uint64) error {
	update := s.Window[s.Index] + gas
	if update > maxBlockGas {
		return errorsmod.Wrapf(ErrMaxGasExceeded, "gas %d > max %d", update, maxBlockGas)
	}

	s.Window[s.Index] = update
	return nil
}

// IncrementHeight increments the current height of the state.
func (s *State) IncrementHeight() {
	s.Index = (s.Index + 1) % uint64(len(s.Window))
	s.Window[s.Index] = 0
}

// UpdateBaseGasPrice updates the learning rate and base gas price based on the AIMD
// learning rate adjustment algorithm. The learning rate is updated
// based on the average gas of the block window. The base gas price is
// update using the new learning rate. Please see the EIP-1559 specification
// for more details.
func (s *State) UpdateBaseGasPrice(logger log.Logger, params Params, maxBlockGas uint64) (gasPrice math.LegacyDec) {
	// Panic catch in case there is an overflow
	defer func() {
		if rec := recover(); rec != nil {
			logger.Error("Panic recovered in state.UpdateBaseGasPrice", "err", rec)
			s.BaseGasPrice = params.MinBaseGasPrice
			gasPrice = s.BaseGasPrice
		}
	}()
	// Calculate the new base gasPrice with the learning rate adjustment.
	currentBlockGas := math.LegacyNewDecFromInt(math.NewIntFromUint64(s.Window[s.Index]))
	targetBlockGas := math.LegacyNewDecFromInt(math.NewIntFromUint64(GetTargetBlockGas(maxBlockGas, params)))
	avgGas := (currentBlockGas.Sub(targetBlockGas)).Quo(targetBlockGas)

	// Truncate the learning rate adjustment to an integer.
	//
	// This is equivalent to
	// 1 + (learningRate * (currentBlockGas - targetBlockGas) / targetBlockGas)
	learningRateAdjustment := math.LegacyOneDec().Add(s.LearningRate.Mul(avgGas))

	// Update the base gasPrice.
	gasPrice = s.BaseGasPrice.Mul(learningRateAdjustment)

	// Ensure the base gasPrice is greater than the minimum base gasPrice.
	if gasPrice.LT(params.MinBaseGasPrice) {
		gasPrice = params.MinBaseGasPrice
	}

	s.BaseGasPrice = gasPrice
	return s.BaseGasPrice
}

// UpdateLearningRate updates the learning rate based on the AIMD
// learning rate adjustment algorithm. The learning rate is updated
// based on the average gas of the block window. There are
// two cases that can occur:
//
//  1. The average gas is above the target threshold. In this
//     case, the learning rate is increased by the alpha parameter. This occurs
//     when blocks are nearly full or empty.
//  2. The average gas is below the target threshold. In this
//     case, the learning rate is decreased by the beta parameter. This occurs
//     when blocks are relatively close to the target block gas.
//
// For more details, please see the EIP-1559 specification.
func (s *State) UpdateLearningRate(params Params, maxBlockGas uint64) (lr math.LegacyDec) {
	// Panic catch in case there is an overflow
	defer func() {
		if rec := recover(); rec != nil {
			s.LearningRate = params.MinLearningRate
			lr = s.LearningRate
		}
	}()

	// Calculate the average gas of the block window.
	avg := s.GetAverageGas(maxBlockGas)

	// Determine if the average gas is above or below the target
	// threshold and adjust the learning rate accordingly.
	if avg.LTE(params.Gamma) || avg.GTE(math.LegacyOneDec().Sub(params.Gamma)) {
		lr = params.Alpha.Add(s.LearningRate)
		if lr.GT(params.MaxLearningRate) {
			lr = params.MaxLearningRate
		}
	} else {
		lr = s.LearningRate.Mul(params.Beta)
		if lr.LT(params.MinLearningRate) {
			lr = params.MinLearningRate
		}
	}

	// Update the current learning rate.
	s.LearningRate = lr
	return s.LearningRate
}

// GetNetGas returns the net gas of the block window.
func (s *State) GetNetGas(maxBlockGas uint64, params Params) math.Int {
	net := math.NewInt(0)

	targetGas := math.NewIntFromUint64(GetTargetBlockGas(maxBlockGas, params))
	for _, gas := range s.Window {
		diff := math.NewIntFromUint64(gas).Sub(targetGas)
		net = net.Add(diff)
	}

	return net
}

// GetAverageGas returns the average gas of the block
// window.
func (s *State) GetAverageGas(maxBlockGas uint64) math.LegacyDec {
	var total uint64
	for _, gas := range s.Window {
		total += gas
	}

	sum := math.LegacyNewDecFromInt(math.NewIntFromUint64(total))

	multiple := math.LegacyNewDecFromInt(math.NewIntFromUint64(uint64(len(s.Window))))
	divisor := math.LegacyNewDecFromInt(math.NewIntFromUint64(maxBlockGas)).Mul(multiple)

	return sum.Quo(divisor)
}

// ValidateBasic performs basic validation on the state.
func (s *State) ValidateBasic() error {
	if s.Window == nil {
		return fmt.Errorf("block gas window cannot be nil or empty")
	}

	if s.BaseGasPrice.IsNil() || s.BaseGasPrice.LTE(math.LegacyZeroDec()) {
		return fmt.Errorf("base gas price must be positive")
	}

	if s.LearningRate.IsNil() || s.LearningRate.LTE(math.LegacyZeroDec()) {
		return fmt.Errorf("learning rate must be positive")
	}

	return nil
}

func GetTargetBlockGas(maxBlockGas uint64, params Params) uint64 {
	targetBlockUtilization := params.TargetBlockUtilization
	return uint64(math.LegacyNewDecFromInt(math.NewIntFromUint64(maxBlockGas)).Mul(targetBlockUtilization).TruncateInt().Int64())
}
