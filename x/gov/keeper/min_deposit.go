package keeper

import (
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// GetActiveProposalsNumber gets the number of active proposals
func (keeper Keeper) GetActiveProposalsNumber(ctx sdk.Context) (activeProposalsNumber uint64) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(types.ActiveProposalsNumberKey)
	if bz == nil {
		return 0
	}

	activeProposalsNumber = types.GetActiveProposalsNumberFromBytes(bz)
	return activeProposalsNumber
}

// SetActiveProposalsNumber sets the new number of active proposals to the store
func (keeper Keeper) SetActiveProposalsNumber(ctx sdk.Context, activeProposalsNumber uint64) {
	store := ctx.KVStore(keeper.storeKey)
	store.Set(types.ActiveProposalsNumberKey, types.GetActiveProposalsNumberBytes(activeProposalsNumber))
}

// IncrementActiveProposalsNumber increments the number of active proposals by one
func (keeper Keeper) IncrementActiveProposalsNumber(ctx sdk.Context) {
	activeProposalsNumber := keeper.GetActiveProposalsNumber(ctx) + 1
	keeper.SetActiveProposalsNumber(ctx, activeProposalsNumber)

	currMinDeposit := keeper.GetMinDeposit(ctx)
	keeper.SetLastMinDeposit(ctx, currMinDeposit, ctx.BlockTime())
}

// DecrementActiveProposalsNumber decrements the number of active proposals by one
func (keeper Keeper) DecrementActiveProposalsNumber(ctx sdk.Context) {
	currentActiveProposalsNumber := keeper.GetActiveProposalsNumber(ctx)
	if currentActiveProposalsNumber > 0 {
		activeProposalsNumber := currentActiveProposalsNumber - 1
		keeper.SetActiveProposalsNumber(ctx, activeProposalsNumber)
	} else {
		panic("number of active proposals should never be negative")
	}

	currMinDeposit := keeper.GetMinDeposit(ctx)
	keeper.SetLastMinDeposit(ctx, currMinDeposit, ctx.BlockTime())
}

// SetLastMinDeposit updates the last min deposit and last min deposit time.
// Used to record these values the last time the number of active proposals changed
func (keeper Keeper) SetLastMinDeposit(ctx sdk.Context, minDeposit sdk.Coins, timeStamp time.Time) {
	store := ctx.KVStore(keeper.storeKey)
	lastMinDeposit := v1.LastMinDeposit{
		Value: minDeposit,
		Time:  &timeStamp,
	}
	bz := keeper.cdc.MustMarshal(&lastMinDeposit)
	store.Set(types.LastMinDepositKey, bz)
}

// GetLastMinDeposit returns the last min deposit and the time it was set
func (keeper Keeper) GetLastMinDeposit(ctx sdk.Context) (sdk.Coins, time.Time) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(types.LastMinDepositKey)
	if bz == nil {
		return sdk.Coins{}, time.Time{}
	}
	var lastMinDeposit v1.LastMinDeposit
	keeper.cdc.MustUnmarshal(bz, &lastMinDeposit)
	return lastMinDeposit.Value, *lastMinDeposit.Time
}

// GetMinDeposit returns the (dynamic) minimum deposit currently required for a proposal
func (keeper Keeper) GetMinDeposit(ctx sdk.Context) sdk.Coins {
	params := keeper.GetParams(ctx)
	minDepositFloor := sdk.Coins(params.MinDepositThrottler.FloorValue)
	tick := params.MinDepositThrottler.UpdatePeriod
	targetActiveProposals := math.NewIntFromUint64(params.MinDepositThrottler.TargetActiveProposals)
	k := params.MinDepositThrottler.SensitivityTargetDistance
	var a sdk.Dec

	numActiveProposals := math.NewIntFromUint64(keeper.GetActiveProposalsNumber(ctx))
	lastMinDeposit, lastMinDepositTime := keeper.GetLastMinDeposit(ctx)
	// get number of ticks passed since last update
	ticksPassed := ctx.BlockTime().Sub(lastMinDepositTime).Nanoseconds() / tick.Nanoseconds()

	if numActiveProposals.GT(targetActiveProposals) {
		a = sdk.MustNewDecFromStr(params.MinDepositThrottler.IncreaseRatio)
	} else {
		a = sdk.MustNewDecFromStr(params.MinDepositThrottler.DecreaseRatio)
	}

	distance := numActiveProposals.Sub(targetActiveProposals)
	percChange := math.LegacyOneDec()
	if !distance.IsZero() {
		b, err := distance.ToLegacyDec().ApproxRoot(k)
		if err != nil {
			// in case of error bypass the sensitivity, i.e. assume k = 1
			b = distance.ToLegacyDec()
		}
		c := a.Mul(b)
		percChange = percChange.Add(c)
	}
	if ticksPassed != 0 {
		percChange = percChange.Power(uint64(ticksPassed))
	}

	minDeposit := sdk.Coins{}
	minDepositFloorDenomsSeen := make(map[string]bool)
	for _, lastMinDepositCoin := range lastMinDeposit {
		minDepositFloorCoinAmt := minDepositFloor.AmountOf(lastMinDepositCoin.Denom)
		if minDepositFloorCoinAmt.IsZero() {
			// minDepositFloor was changed and this coin was removed,
			// reflect this also in the current min deposit, i.e. remove
			// this coin
			continue
		}
		minDepositFloorDenomsSeen[lastMinDepositCoin.Denom] = true
		minDepositCoinAmt := lastMinDepositCoin.Amount.ToLegacyDec().Mul(percChange).TruncateInt()
		if minDepositCoinAmt.LT(minDepositFloorCoinAmt) {
			minDeposit = append(minDeposit, sdk.NewCoin(lastMinDepositCoin.Denom, minDepositFloorCoinAmt))
		} else {
			minDeposit = append(minDeposit, sdk.NewCoin(lastMinDepositCoin.Denom, minDepositCoinAmt))
		}
	}

	// make sure any new denoms in minDepositFloor are added to minDeposit
	for _, minDepositFloorCoin := range minDepositFloor {
		if _, seen := minDepositFloorDenomsSeen[minDepositFloorCoin.Denom]; !seen {
			minDepositCoinAmt := minDepositFloorCoin.Amount.ToLegacyDec().Mul(percChange).TruncateInt()
			if minDepositCoinAmt.LT(minDepositFloorCoin.Amount) {
				minDeposit = append(minDeposit, minDepositFloorCoin)
			} else {
				minDeposit = append(minDeposit, sdk.NewCoin(minDepositFloorCoin.Denom, minDepositCoinAmt))
			}
		}
	}

	return minDeposit
}
