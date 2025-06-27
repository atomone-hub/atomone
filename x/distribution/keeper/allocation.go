package keeper

import (
	"context"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AllocateTokens performs reward and fee distribution to all validators
// based on the F1 fee distribution specification and ADR-004 (Nakamoto Bonus).
func (k Keeper) AllocateTokens(ctx context.Context, totalPreviousPower int64, bondedVotes []abci.VoteInfo) error {
	// Step 1: fetch and clear the collected fees for distribution, since this is
	// called in BeginBlock, collected fees will be from the previous block
	// (and distributed to the previous proposer)
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)

	// transfer collected fees to the distribution module account
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt)
	if err != nil {
		return err
	}

	// temporary workaround to keep CanWithdrawInvariant happy
	// general discussions here: https://github.com/cosmos/cosmos-sdk/issues/2906#issuecomment-441867634
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	if totalPreviousPower == 0 {
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		return k.FeePool.Set(ctx, feePool)
	}

	/// Step 2: calculate fraction allocated to validators
	communityTax, err := k.GetCommunityTax(ctx)
	if err != nil {
		return err
	}

	voteMultiplier := math.LegacyOneDec().Sub(communityTax)
	feeMultiplier := feesCollected.MulDecTruncate(voteMultiplier)

	// Step 3: Get bonded validators and total bonded power
	bondedValidators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return err
	}
	totalBonded := math.ZeroInt()
	for _, val := range bondedValidators {
		totalBonded = totalBonded.Add(val.BondedTokens())
	}

	// Step 4: Get block provision (minted tokens)
	blockProvision, err := k.BlockProvision(ctx)
	if err != nil {
		return err
	}

	// Step 5: Get eta (Nakamoto Bonus coefficient)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	eta := params.NakamotoBonusCoefficient

	// Step 6: Split blockProvision using Nakamoto Bonus logic
	rewardMap := k.SplitReward(
		blockProvision,
		eta,
		bondedValidators,
		totalBonded,
	)

	// Step 7: Distribute rewards (fees + mint) to validators
	remaining := feeMultiplier // Start with fees to allocate

	for _, vote := range bondedVotes {
		validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		if err != nil {
			return err
		}
		valAddr := validator.GetOperator()

		// Calculate fee reward (proportional by voting power)
		powerFraction := math.LegacyNewDec(vote.Validator.Power).QuoTruncate(math.LegacyNewDec(totalPreviousPower))
		feeReward := feeMultiplier.MulDecTruncate(powerFraction)

		// Get block reward (PR + NB) for this validator
		blockReward := rewardMap[valAddr] // This is sdk.DecCoins

		// Total reward for this block for this validator
		totalReward := feeReward.Add(blockReward...)

		// Allocate (handles commission, outstanding, etc)
		err = k.AllocateTokensToValidator(ctx, validator, totalReward)
		if err != nil {
			return err
		}

		remaining = remaining.Sub(feeReward)
	}

	// Send any leftover fees (from rounding) to community pool
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
// - totalReward: the total reward for the block
// - eta: the nakamoto bonus coefficient (e.g., 0.05 means 5% NB, 95% PR)
// - bondedValidators: the list of bonded validators for this block
// - totalBonded: total stake bonded across all validators
func (k Keeper) SplitReward(
	totalReward sdk.DecCoin,
	eta math.LegacyDec,
	bondedValidators []stakingtypes.Validator,
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
