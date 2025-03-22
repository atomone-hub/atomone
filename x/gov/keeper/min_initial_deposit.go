package keeper

import (
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// GetInactiveProposalsNumber gets the number of inactive proposals
func (keeper Keeper) GetInactiveProposalsNumber(ctx sdk.Context) (inactiveProposalsNumber uint64) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(types.InactiveProposalsNumberKey)
	if bz == nil {
		return 0
	}

	inactiveProposalsNumber = types.GetInactiveProposalsNumberFromBytes(bz)
	return inactiveProposalsNumber
}

// SetInactiveProposalsNumber sets the new number of inactive proposals to the store
func (keeper Keeper) SetInactiveProposalsNumber(ctx sdk.Context, inactiveProposalsNumber uint64) {
	store := ctx.KVStore(keeper.storeKey)
	store.Set(types.InactiveProposalsNumberKey, types.GetInactiveProposalsNumberBytes(inactiveProposalsNumber))
}

// IncrementInactiveProposalsNumber increments the number of inactive proposals by one
func (keeper Keeper) IncrementInactiveProposalsNumber(ctx sdk.Context) {
	inactiveProposalsNumber := keeper.GetInactiveProposalsNumber(ctx) + 1
	keeper.SetInactiveProposalsNumber(ctx, inactiveProposalsNumber)

	currMinInitialDeposit := keeper.GetMinInitialDeposit(ctx)
	keeper.SetLastMinInitialDeposit(ctx, currMinInitialDeposit, ctx.BlockTime())
}

// DecrementInactiveProposalsNumber decrements the number of inactive proposals by one
func (keeper Keeper) DecrementInactiveProposalsNumber(ctx sdk.Context) {
	currentInactiveProposalsNumber := keeper.GetInactiveProposalsNumber(ctx)
	if currentInactiveProposalsNumber > 0 {
		inactiveProposalsNumber := currentInactiveProposalsNumber - 1
		keeper.SetInactiveProposalsNumber(ctx, inactiveProposalsNumber)
	} else {
		panic("number of inactive proposals should never be negative")
	}

	currMinInitialDeposit := keeper.GetMinInitialDeposit(ctx)
	keeper.SetLastMinInitialDeposit(ctx, currMinInitialDeposit, ctx.BlockTime())
}

// SetLastMinInitialDeposit updates the last min initial deposit and last min
// initial deposit time.
// Used to record these values the last time the number of inactive proposals changed
func (keeper Keeper) SetLastMinInitialDeposit(ctx sdk.Context, minInitialDeposit sdk.Coins, timeStamp time.Time) {
	store := ctx.KVStore(keeper.storeKey)
	lastMinInitialDeposit := v1.LastMinDeposit{
		Value: minInitialDeposit,
		Time:  &timeStamp,
	}
	bz := keeper.cdc.MustMarshal(&lastMinInitialDeposit)
	store.Set(types.LastMinInitialDepositKey, bz)
}

// GetLastMinInitialDeposit returns the last min initial deposit and the time it was set
func (keeper Keeper) GetLastMinInitialDeposit(ctx sdk.Context) (sdk.Coins, time.Time) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(types.LastMinInitialDepositKey)
	if bz == nil {
		return sdk.Coins{}, time.Time{}
	}
	var lastMinInitialDeposit v1.LastMinDeposit
	keeper.cdc.MustUnmarshal(bz, &lastMinInitialDeposit)
	return lastMinInitialDeposit.Value, *lastMinInitialDeposit.Time
}

// GetMinInitialDeposit returns the (dynamic) minimum initial deposit currently required for
// proposal submission
func (keeper Keeper) GetMinInitialDeposit(ctx sdk.Context) sdk.Coins {
	logger := keeper.Logger(ctx)

	params := keeper.GetParams(ctx)
	minInitialDepositFloor := sdk.Coins(params.MinInitialDepositThrottler.FloorValue)
	tick := params.MinInitialDepositThrottler.UpdatePeriod
	targetInactiveProposals := math.NewIntFromUint64(params.MinInitialDepositThrottler.TargetProposals)
	k := params.MinInitialDepositThrottler.SensitivityTargetDistance
	var a sdk.Dec

	numInactiveProposals := math.NewIntFromUint64(keeper.GetInactiveProposalsNumber(ctx))
	lastMinInitialDeposit, lastMinInitialDepositTime := keeper.GetLastMinInitialDeposit(ctx)
	// get number of ticks passed since last update
	ticksPassed := ctx.BlockTime().Sub(lastMinInitialDepositTime).Nanoseconds() / tick.Nanoseconds()

	if numInactiveProposals.GT(targetInactiveProposals) {
		a = sdk.MustNewDecFromStr(params.MinInitialDepositThrottler.IncreaseRatio)
	} else {
		a = sdk.MustNewDecFromStr(params.MinInitialDepositThrottler.DecreaseRatio)
	}

	distance := numInactiveProposals.Sub(targetInactiveProposals)
	percChange := math.LegacyOneDec()
	if !distance.IsZero() {
		// ApproxRoot is here being called on a relatively small positive integer
		// with a value of k expected to also be relatively small.
		// This is a safe operation and should not error.
		b, err := distance.ToLegacyDec().ApproxRoot(k)
		if err != nil {
			// in case of error bypass the sensitivity, i.e. assume k = 1
			b = distance.ToLegacyDec()
			logger.Error("failed to calculate ApproxRoot", "error", err)
		}
		c := a.Mul(b)
		percChange = percChange.Add(c)
	}
	if ticksPassed != 0 {
		percChange = percChange.Power(uint64(ticksPassed))
	}

	minInitialDeposit := sdk.Coins{}
	minInitialDepositFloorDenomsSeen := make(map[string]bool)
	for _, lastMinInitialDepositCoin := range lastMinInitialDeposit {
		minInitialDepositFloorCoinAmt := minInitialDepositFloor.AmountOf(lastMinInitialDepositCoin.Denom)
		if minInitialDepositFloorCoinAmt.IsZero() {
			// minInitialDepositFloor was changed and this coin was removed,
			// reflect this also in the current min initial deposit,
			// i.e. remove this coin
			continue
		}
		minInitialDepositFloorDenomsSeen[lastMinInitialDepositCoin.Denom] = true
		minInitialDepositCoinAmt := lastMinInitialDepositCoin.Amount.ToLegacyDec().Mul(percChange).TruncateInt()
		if minInitialDepositCoinAmt.LT(minInitialDepositFloorCoinAmt) {
			minInitialDeposit = append(minInitialDeposit, sdk.NewCoin(lastMinInitialDepositCoin.Denom, minInitialDepositFloorCoinAmt))
		} else {
			minInitialDeposit = append(minInitialDeposit, sdk.NewCoin(lastMinInitialDepositCoin.Denom, minInitialDepositCoinAmt))
		}
	}

	// make sure any new denoms in minInitialDepositFloor are added to minInitialDeposit
	for _, minInitialDepositFloorCoin := range minInitialDepositFloor {
		if _, seen := minInitialDepositFloorDenomsSeen[minInitialDepositFloorCoin.Denom]; !seen {
			minInitialDepositCoinAmt := minInitialDepositFloorCoin.Amount.ToLegacyDec().Mul(percChange).TruncateInt()
			if minInitialDepositCoinAmt.LT(minInitialDepositFloorCoin.Amount) {
				minInitialDeposit = append(minInitialDeposit, minInitialDepositFloorCoin)
			} else {
				minInitialDeposit = append(minInitialDeposit, sdk.NewCoin(minInitialDepositFloorCoin.Denom, minInitialDepositCoinAmt))
			}
		}
	}

	return minInitialDeposit
}
