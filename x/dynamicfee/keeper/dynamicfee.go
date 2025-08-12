package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/dynamicfee/types"
)

// UpdateDynamicfee updates the base fee and learning rate based on the
// AIMD learning rate adjustment algorithm. Note that if the dynamic fee
// pricing is disabled, this function will return without updating the
// dynamic fee pricing. This is executed in EndBlock which allows the next
// block's base fee to be readily available for wallets to estimate gas prices.
func (k *Keeper) UpdateDynamicfee(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := k.Logger(sdkCtx)

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	logger.Info(
		"updated the dynamic fee pricing",
		"params", params,
	)

	if !params.Enabled {
		return nil
	}

	maxBlockGas := k.GetMaxBlockGas(ctx, params)

	state, err := k.GetState(ctx)
	if err != nil {
		return err
	}

	// Update the learning rate based on the block gas seen in the
	// current block. This is the AIMD learning rate adjustment algorithm.
	newLR := state.UpdateLearningRate(params, maxBlockGas)

	// Update the base gas price based with the new learning rate.
	newBaseGasPrice := state.UpdateBaseGasPrice(logger, params, maxBlockGas)

	logger.Info(
		"updated the dynamic fee pricing",
		"height", sdkCtx.BlockHeight(),
		"new_base_gas_price", newBaseGasPrice,
		"new_learning_rate", newLR,
		"average_block_gas", state.GetAverageGas(maxBlockGas),
		"net_block_gas", state.GetNetGas(maxBlockGas, params),
	)

	// Increment the height of the state and set the new state.
	state.IncrementHeight()
	return k.SetState(ctx, state)
}

// GetMaxBlockGas returns the maximum gas of a block
// It returns the value obtained from ConsensusParams if
// it is different from 0 or -1, otherwise it returns
// DefaultMaxBlockGas
func (k *Keeper) GetMaxBlockGas(ctx context.Context, params types.Params) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	maxBlockGas := sdkCtx.ConsensusParams().Block.GetMaxGas()
	if maxBlockGas == 0 || maxBlockGas == -1 {
		return params.DefaultMaxBlockGas
	}
	return uint64(maxBlockGas)
}

// GetBaseGasPrice returns the base fee from the dynamic fee pricing state.
func (k *Keeper) GetBaseGasPrice(ctx context.Context) (math.LegacyDec, error) {
	state, err := k.GetState(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return state.BaseGasPrice, nil
}

// GetLearningRate returns the learning rate from the dynamic fee pricing state.
func (k *Keeper) GetLearningRate(ctx context.Context) (math.LegacyDec, error) {
	state, err := k.GetState(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return state.LearningRate, nil
}

// GetMinGasPrice returns the mininum gas prices for given denom as
// sdk.DecCoins from the dynamic fee pricing state.
func (k *Keeper) GetMinGasPrice(ctx context.Context, denom string) (sdk.DecCoin, error) {
	baseGasPrice, err := k.GetBaseGasPrice(ctx)
	if err != nil {
		return sdk.DecCoin{}, err
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return sdk.DecCoin{}, err
	}

	var gasPrice sdk.DecCoin

	if params.FeeDenom == denom {
		gasPrice = sdk.NewDecCoinFromDec(params.FeeDenom, baseGasPrice)
	} else {
		gasPrice, err = k.ResolveToDenom(ctx, sdk.NewDecCoinFromDec(params.FeeDenom, baseGasPrice), denom)
		if err != nil {
			return sdk.DecCoin{}, err
		}
	}

	return gasPrice, nil
}

// GetMinGasPrices returns the mininum gas prices as sdk.DecCoins from the
// dynamic fee pricing state.
func (k *Keeper) GetMinGasPrices(ctx context.Context) (sdk.DecCoins, error) {
	baseGasPrice, err := k.GetBaseGasPrice(ctx)
	if err != nil {
		return sdk.NewDecCoins(), err
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return sdk.NewDecCoins(), err
	}

	minGasPrice := sdk.NewDecCoinFromDec(params.FeeDenom, baseGasPrice)
	minGasPrices := sdk.NewDecCoins(minGasPrice)

	extraDenoms, err := k.resolver.ExtraDenoms(ctx)
	if err != nil {
		return sdk.NewDecCoins(), err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, denom := range extraDenoms {
		gasPrice, err := k.ResolveToDenom(ctx, minGasPrice, denom)
		if err != nil {
			k.Logger(sdkCtx).Info(
				"failed to convert gas price",
				"min gas price", minGasPrice,
				"denom", denom,
			)
			continue
		}
		minGasPrices = minGasPrices.Add(gasPrice)
	}

	return minGasPrices, nil
}
