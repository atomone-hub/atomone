package keeper

import (
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/distribution/types"
)

const (
	EtaUpdateInterval = 120_000 // every 120k blocks (~1 week)
	EtaStep           = 3       // step to increase or decrease η
)

// AdjustEta is called to adjust η dynamically for each block.
func (k Keeper) AdjustEta(ctx sdk.Context) error {
	if ctx.BlockHeight()%EtaUpdateInterval != 0 {
		return nil
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	if !params.NakamotoBonusEnabled {
		// Always set eta to zero and skip dynamic update
		if params.NakamotoBonusCoefficient.IsZero() {
			// Already zero, nothing to do
			return nil
		}
		params.NakamotoBonusCoefficient = math.LegacyZeroDec()
		return k.Params.Set(ctx, params)
	}

	validators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return err
	}
	n := len(validators)
	if n < 3 {
		return nil
	}
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].GetBondedTokens().GT(validators[j].GetBondedTokens())
	})

	var high, low []stakingtypes.Validator
	if n < 100 {
		third := n / 3
		high = validators[:third]
		low = validators[n-third:]
	} else {
		high = validators[:33]
		low = validators[66:]
	}

	sum := func(vals []stakingtypes.Validator) math.Int {
		total := math.ZeroInt()
		for _, v := range vals {
			total = total.Add(v.GetBondedTokens())
		}
		return total
	}
	avg := func(vals []stakingtypes.Validator) math.LegacyDec {
		if len(vals) == 0 {
			return math.LegacyZeroDec()
		}
		return math.LegacyNewDecFromInt(sum(vals)).QuoInt64(int64(len(vals)))
	}
	highAvg := avg(high)
	lowAvg := avg(low)
	eta := params.NakamotoBonusCoefficient

	if lowAvg.IsZero() || highAvg.Quo(lowAvg).GTE(math.LegacyNewDec(EtaStep)) {
		eta = eta.Add(math.LegacyNewDecWithPrec(EtaStep, 2))
	} else {
		eta = eta.Sub(math.LegacyNewDecWithPrec(EtaStep, 2))
	}
	if eta.LT(math.LegacyZeroDec()) {
		eta = math.LegacyZeroDec()
	}
	if eta.GT(math.LegacyOneDec()) {
		eta = math.LegacyOneDec()
	}

	if !eta.Equal(params.NakamotoBonusCoefficient) {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeNakamotoCoefficient,
				sdk.NewAttribute(types.AttributeNakamotoCoefficient, eta.String()),
			),
		)
	}

	params.NakamotoBonusCoefficient = eta
	return k.Params.Set(ctx, params)
}
