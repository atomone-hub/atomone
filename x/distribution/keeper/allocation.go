package keeper

import (
	"context"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k Keeper) AllocateTokens(ctx context.Context, totalPreviousPower int64, bondedVotes []abci.VoteInfo) error {
	// Get collected fees from the fee collector module account
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)

	// Transfer collected fees to the distribution module
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt)
	if err != nil {
		return err
	}

	// Get the current fee pool
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	// If there's no validator power, redirect everything to the community pool
	if totalPreviousPower == 0 {
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		return k.FeePool.Set(ctx, feePool)
	}

	// Get community tax and Nakamoto bonus ratio η
	communityTax, err := k.GetCommunityTax(ctx)
	if err != nil {
		return err
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	nakamotoCoefficient := params.NakamotoBonusCoefficient // the nakamoto bonus coefficient (e.g., 0.05 means 5% NB, 95% PR)

	// Compute total validator rewards (after community tax)
	voteMultiplier := math.LegacyOneDec().Sub(communityTax)
	validatorTotalReward := feesCollected.MulDecTruncate(voteMultiplier)

	// Split reward into Proportional (PR_i) and Nakamoto Bonus (NB_i)
	nakamotoBonus := validatorTotalReward.MulDecTruncate(nakamotoCoefficient)
	proportionalReward := validatorTotalReward.Sub(nakamotoBonus)

	// Compute per-validator fixed Nakamoto bonus
	numValidators := int64(len(bondedVotes))
	nbPerValidator := sdk.NewDecCoinFromDec("uatom", math.LegacyZeroDec())
	if numValidators > 0 && len(nakamotoBonus) > 0 {
		denom := nakamotoBonus[0].Denom
		amount := nakamotoBonus.AmountOf(denom).Quo(math.LegacyNewDec(numValidators))
		nbPerValidator = sdk.NewDecCoinFromDec(denom, amount)
	}

	remaining := feesCollected

	// Distribute rewards to each validator
	for _, vote := range bondedVotes {
		validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		if err != nil {
			return err
		}

		// Compute proportional share based on voting power
		powerFraction := math.LegacyNewDec(vote.Validator.Power).QuoTruncate(math.LegacyNewDec(totalPreviousPower))
		proportional := proportionalReward.MulDecTruncate(powerFraction)

		// Add fixed Nakamoto bonus to proportional share
		totalReward := proportional.Add(nbPerValidator)

		err = k.AllocateTokensToValidator(ctx, validator, totalReward)
		if err != nil {
			return err
		}

		remaining = remaining.Sub(totalReward)
	}

	// Add any remaining tokens to the community pool
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	return k.FeePool.Set(ctx, feePool)
}

// BlockProvision returns the provisions for a block based on the annual
// provisions rate. Uses mint module's query server.
func (k Keeper) BlockProvision(ctx context.Context) (sdk.DecCoin, error) {
	mintParams, err := k.MintQuery.Params(ctx, nil)
	if err != nil {
		return sdk.DecCoin{}, err
	}

	annualProvisionsResult, err := k.MintQuery.AnnualProvisions(ctx, nil)
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
func (k Keeper) SplitReward(ctx context.Context) (map[string]sdk.DecCoins, error) {
	bondedValidators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return nil, err
	}
	totalBonded := math.ZeroInt()
	for _, val := range bondedValidators {
		totalBonded = totalBonded.Add(val.BondedTokens())
	}

	// Step 4: Get block provision (minted tokens)
	blockProvision, err := k.BlockProvision(ctx)
	if err != nil {
		return nil, err
	}

	// Step 5: Get eta (Nakamoto Bonus coefficient)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	eta := params.NakamotoBonusCoefficient // the nakamoto bonus coefficient (e.g., 0.05 means 5% NB, 95% PR)

	// Calculate PR and NB components
	var (
		proportional  = blockProvision.Amount.Mul(math.LegacyOneDec().Sub(eta))
		nakamotoBonus = blockProvision.Amount.Mul(eta)
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
			sdk.NewDecCoinFromDec(blockProvision.Denom, total),
		)
	}

	return rewards, nil
}

// AllocateTokensToValidator allocate tokens to a particular validator,
// splitting according to commission.
func (k Keeper) AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error {
	// split tokens between validator and delegators according to commission
	commission := tokens.MulDec(val.GetCommission())
	shared := tokens.Sub(commission)

	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return err
	}

	// update current commission
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCommission,
			sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator()),
		),
	)
	currentCommission, err := k.GetValidatorAccumulatedCommission(ctx, valBz)
	if err != nil {
		return err
	}

	currentCommission.Commission = currentCommission.Commission.Add(commission...)
	err = k.SetValidatorAccumulatedCommission(ctx, valBz, currentCommission)
	if err != nil {
		return err
	}

	// update current rewards
	currentRewards, err := k.GetValidatorCurrentRewards(ctx, valBz)
	if err != nil {
		return err
	}

	currentRewards.Rewards = currentRewards.Rewards.Add(shared...)
	err = k.SetValidatorCurrentRewards(ctx, valBz, currentRewards)
	if err != nil {
		return err
	}

	// update outstanding rewards
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, tokens.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator()),
		),
	)

	outstanding, err := k.GetValidatorOutstandingRewards(ctx, valBz)
	if err != nil {
		return err
	}

	outstanding.Rewards = outstanding.Rewards.Add(tokens...)
	return k.SetValidatorOutstandingRewards(ctx, valBz, outstanding)
}
