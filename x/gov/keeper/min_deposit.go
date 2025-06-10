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
	logger := keeper.Logger(ctx)

	params := keeper.GetParams(ctx)
	tick := params.MinDepositThrottler.UpdatePeriod
	lastMinDeposit, lastMinDepositTime := keeper.GetLastMinDeposit(ctx)
	if checkElapsedTime && ctx.BlockTime().Sub(lastMinDepositTime).Nanoseconds() < tick.Nanoseconds() {
		return
	}

	minDepositFloor := sdk.Coins(params.MinDepositThrottler.FloorValue)
	targetActiveProposals := math.NewIntFromUint64(params.MinDepositThrottler.TargetActiveProposals)
	k := params.MinDepositThrottler.DecreaseSensitivityTargetDistance
	var alpha math.LegacyDec

	numActiveProposals := math.NewIntFromUint64(keeper.GetActiveProposalsNumber(ctx))
	if numActiveProposals.GTE(targetActiveProposals) {
		if checkElapsedTime {
			// no time-based increases
			return
		}
		alpha = math.LegacyMustNewDecFromStr(params.MinDepositThrottler.IncreaseRatio)
	} else {
		distance := numActiveProposals.Sub(targetActiveProposals)
		if !checkElapsedTime {
			// decreases can only happen due to time-based updates
			// and if the number of active proposals is below the target
			return
		}
		alpha = math.LegacyMustNewDecFromStr(params.MinDepositThrottler.DecreaseRatio)
		// ApproxRoot is here being called on a relatively small positive
		// integer (when distance < 0, ApproxRoot will return
		// `|distance|.ApproxRoot(k) * -1`) with a value of k expected to also
		// be relatively small (<= 100).
		// This is a safe operation and should not error.
		b, err := distance.ToLegacyDec().ApproxRoot(k)
		if err != nil {
			// in case of error bypass the sensitivity, i.e. assume k = 1
			b = distance.ToLegacyDec()
			logger.Error("failed to calculate ApproxRoot for min deposit",
				"error", err,
				"distance", distance.String(),
				"k", k,
				"fallback", "using k=1")
		}
		alpha = alpha.Mul(b)
	}
	percChange := math.LegacyOneDec().Add(alpha)
	newMinDeposit := v1.GetNewMinDeposit(minDepositFloor, lastMinDeposit, percChange)
	keeper.SetLastMinDeposit(ctx, newMinDeposit, ctx.BlockTime())

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMinDepositChange,
			sdk.NewAttribute(types.AttributeKeyNewMinDeposit, newMinDeposit.String()),
			sdk.NewAttribute(types.AttributeKeyLastMinDeposit, lastMinDeposit.String()),
		),
	)
}
