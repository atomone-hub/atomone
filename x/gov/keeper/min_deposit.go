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

	keeper.UpdateMinDeposit(ctx, false)
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

	keeper.UpdateMinDeposit(ctx, false)
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
	minDeposit, _ := keeper.GetLastMinDeposit(ctx)

	if minDeposit.Empty() {
		// ValidateBasic prevents an empty FloorValue
		// (and thus an empty deposit), if LastMinDeposit is empty
		// it means it was never set, so we return the floor value
		return keeper.GetParams(ctx).MinDepositThrottler.GetFloorValue()
	}

	return minDeposit
}

// UpdateMinDeposit updates the minimum deposit required for a proposal
func (keeper Keeper) UpdateMinDeposit(ctx sdk.Context, checkElapsedTime bool) {
	params := keeper.GetParams(ctx)
	tick := params.MinDepositThrottler.UpdatePeriod
	lastMinDeposit, lastMinDepositTime := keeper.GetLastMinDeposit(ctx)
	if checkElapsedTime && ctx.BlockTime().Sub(lastMinDepositTime).Nanoseconds() < tick.Nanoseconds() {
		return
	}

	minDepositFloor := sdk.Coins(params.MinDepositThrottler.FloorValue)
	targetActiveProposals := math.NewIntFromUint64(params.MinDepositThrottler.TargetActiveProposals)
	k := params.MinDepositThrottler.SensitivityTargetDistance
	var a sdk.Dec

	numActiveProposals := math.NewIntFromUint64(keeper.GetActiveProposalsNumber(ctx))
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

	newMinDeposit := sdk.Coins{}
	minDepositFloorDenomsSeen := make(map[string]bool)
	for _, lastMinDepositCoin := range lastMinDeposit {
		minDepositFloorCoinAmt := minDepositFloor.AmountOf(lastMinDepositCoin.Denom)
		if minDepositFloorCoinAmt.IsZero() {
			// minDepositFloor was changed since last update,
			// and this coin was removed.
			// reflect this also in the current min initial deposit,
			// i.e. remove this coin
			continue
		}
		minDepositFloorDenomsSeen[lastMinDepositCoin.Denom] = true
		minDepositCoinAmt := lastMinDepositCoin.Amount.ToLegacyDec().Mul(percChange).TruncateInt()
		if minDepositCoinAmt.LT(minDepositFloorCoinAmt) {
			newMinDeposit = append(newMinDeposit, sdk.NewCoin(lastMinDepositCoin.Denom, minDepositFloorCoinAmt))
		} else {
			newMinDeposit = append(newMinDeposit, sdk.NewCoin(lastMinDepositCoin.Denom, minDepositCoinAmt))
		}
	}

	// make sure any new denoms in minDepositFloor are added to minDeposit
	for _, minDepositFloorCoin := range minDepositFloor {
		if _, seen := minDepositFloorDenomsSeen[minDepositFloorCoin.Denom]; !seen {
			minDepositCoinAmt := minDepositFloorCoin.Amount.ToLegacyDec().Mul(percChange).TruncateInt()
			if minDepositCoinAmt.LT(minDepositFloorCoin.Amount) {
				newMinDeposit = append(newMinDeposit, minDepositFloorCoin)
			} else {
				newMinDeposit = append(newMinDeposit, sdk.NewCoin(minDepositFloorCoin.Denom, minDepositCoinAmt))
			}
		}
	}

	keeper.SetLastMinDeposit(ctx, newMinDeposit, ctx.BlockTime())
}
