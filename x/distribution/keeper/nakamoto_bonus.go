package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k Keeper) DistributeNakamotoReward(ctx context.Context) error {
	totalReward, err := k.BlockProvision(ctx) // or however your chain calculates it
	if err != nil {
		return err
	}

	// Get η parameter (Nakamoto Bonus coefficient)
	params, err := k.Params.Get(ctx) // or however your chain calculates it
	if err != nil {
		return err
	}
	eta := params.NakamotoBonusCoefficient

	// Get all bonded validators and total bonded stake
	bondedValidators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return err
	}

	totalBonded := math.ZeroInt()
	for _, val := range bondedValidators {
		totalBonded = totalBonded.Add(val.BondedTokens())
	}

	// Split the reward according to the Nakamoto Bonus logic
	rewardMap := k.SplitReward(
		totalReward,
		eta,
		bondedValidators,
		totalBonded,
	)

	// Distribute rewards to each validator
	for _, val := range bondedValidators {
		valAddr := val.GetOperator()
		reward := rewardMap[valAddr]

		// Delegate reward distribution logic (existing logic)
		// This typically mints/distributes to validator and delegators
		if err := k.AllocateTokensToValidator(ctx, val, reward); err != nil {
			return err
		}
	}

	return nil
}

// BlockProvision returns the provisions for a block based on the annual
// provisions rate.
func (k Keeper) BlockProvision(ctx context.Context) (sdk.DecCoin, error) {
	mintParams, err := k.mintQuery.Params(ctx, nil)
	if err != nil {
		return sdk.DecCoin{}, err
	}

	annualProvisionsResult, err := k.mintQuery.AnnualProvisions(ctx, nil)
	if err != nil {
		return sdk.DecCoin{}, err
	}

	var (
		annualProvisions = annualProvisionsResult.AnnualProvisions
		blocksPerYear    = int64(mintParams.Params.BlocksPerYear)
		mintDenom        = mintParams.Params.MintDenom
	)
	provisionAmt := annualProvisions.QuoInt(math.NewInt(blocksPerYear))
	return sdk.NewDecCoin(mintDenom, provisionAmt.TruncateInt()), nil
}

// SplitReward splits the total reward into proportional and nakamoto bonus.
// - totalReward: the total reward for the block
// - eta: the nakamoto bonus coefficient (e.g., 0.05 means 5% NB, 95% PR)
// - bondedValidators: the list of bonded validators for this block
// - totalBonded: total stake bonded across all validators
func (k Keeper) SplitReward(
	totalReward sdk.DecCoin,
	eta math.LegacyDec,
	bondedValidators []types.Validator,
	totalBonded math.Int,
) map[string]sdk.DecCoins {
	// Calculate PR and NB components
	var (
		proportional  = totalReward.Amount.Mul(math.LegacyOneDec().Sub(eta))
		nakamotoBonus = totalReward.Amount.Mul(eta)
		numValidators = math.LegacyNewDec(int64(len(bondedValidators)))
		rewards       = make(map[string]sdk.DecCoins)
	)

	for _, val := range bondedValidators {
		stake := val.GetBondedTokens()
		prShare := math.LegacyZeroDec()
		if totalBonded.IsPositive() {
			prShare = math.LegacyNewDecFromInt(stake).QuoInt(totalBonded).Mul(proportional)
		}

		nbShare := math.LegacyZeroDec()
		if numValidators.IsPositive() {
			nbShare = nakamotoBonus.Quo(numValidators)
		}

		total := prShare.Add(nbShare)
		rewards[val.GetOperator()] = sdk.NewDecCoins(
			sdk.NewDecCoinFromDec(totalReward.Denom, total),
		)
	}

	return rewards
}
