package types

import (
	"cosmossdk.io/math"

	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

// Note: We use the same default values as Ethereum for the EIP-1559 dynamic
// fee pricing implementation. These parameters do not implement the AIMD
// learning rate adjustment algorithm.

var (
	// DefaultWindow is the default window size for the sliding window
	// used to calculate the base fee. In the base EIP-1559 implementation,
	// only the previous block is considered.
	DefaultWindow uint64 = 1

	// DefaultAlpha is not used in the base EIP-1559 implementation.
	DefaultAlpha = math.LegacyMustNewDecFromStr("0.0")

	// DefaultBeta is not used in the base EIP-1559 implementation.
	DefaultBeta = math.LegacyMustNewDecFromStr("1.0")

	// DefaultGamma is not used in the base EIP-1559 implementation.
	DefaultGamma = math.LegacyMustNewDecFromStr("0.0")

	// DefaultMinBaseGasPrice is the default minimum base gas price.
	DefaultMinBaseGasPrice = math.LegacyMustNewDecFromStr("0.01")

	// DefaultTargetBlockUtilization is the default target block utilization.
	DefaultTargetBlockUtilization = math.LegacyMustNewDecFromStr("0.5")

	// DefaultMaxBlockGas is the default max block gas that is used
	// when consensus_params.block.max_gas returns 0 or -1
	DefaultMaxBlockGas uint64 = 100_000_000

	// DefaultMinLearningRate is not used in the base EIP-1559 implementation.
	DefaultMinLearningRate = math.LegacyMustNewDecFromStr("0.125")

	// DefaultMaxLearningRate is not used in the base EIP-1559 implementation.
	DefaultMaxLearningRate = math.LegacyMustNewDecFromStr("0.125")

	// DefaultFeeDenom is the Cosmos SDK default bond denom.
	DefaultFeeDenom = photontypes.Denom
)

// DefaultParams returns a default set of parameters that implements
// the EIP-1559 dynamic fee pricing implementation without the AIMD
// learning rate adjustment algorithm.
func DefaultParams() Params {
	return NewParams(
		DefaultWindow,
		DefaultAlpha,
		DefaultBeta,
		DefaultGamma,
		DefaultMinBaseGasPrice,
		DefaultTargetBlockUtilization,
		DefaultMaxBlockGas,
		DefaultMinLearningRate,
		DefaultMaxLearningRate,
		DefaultFeeDenom,
		true,
	)
}

// DefaultState returns the default state for the EIP-1559 dynamic fee pricing
// implementation without the AIMD learning rate adjustment algorithm.
func DefaultState() State {
	return NewState(
		DefaultWindow,
		DefaultMinBaseGasPrice,
		DefaultMinLearningRate,
	)
}

// DefaultGenesisState returns a default genesis state that implements
// the EIP-1559 dynamic fee pricing implementation without the AIMD
// learning rate adjustment algorithm.
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(DefaultParams(), DefaultState())
}
