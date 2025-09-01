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

	keeper.UpdateMinInitialDeposit(ctx, false)
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
	minInitialDeposit, _ := keeper.GetLastMinInitialDeposit(ctx)

	if minInitialDeposit.Empty() {
		// ValidateBasic prevents an empty FloorValue
		// (and thus an empty deposit), if LastMinInitialDeposit is empty
		// it means it was never set, so we return the floor value
		return keeper.GetParams(ctx).MinInitialDepositThrottler.GetFloorValue()
	}

	return minInitialDeposit
}

// UpdateMinInitialDeposit updates the min initial deposit required for proposal submission
func (keeper Keeper) UpdateMinInitialDeposit(ctx sdk.Context, checkElapsedTime bool) {
	logger := keeper.Logger(ctx)

	params := keeper.GetParams(ctx)
	tick := params.MinInitialDepositThrottler.UpdatePeriod
	lastMinInitialDeposit, lastMinInitialDepositTime := keeper.GetLastMinInitialDeposit(ctx)
	if checkElapsedTime && ctx.BlockTime().Sub(lastMinInitialDepositTime).Nanoseconds() < tick.Nanoseconds() {
		return
	}

	minInitialDepositFloor := sdk.Coins(params.MinInitialDepositThrottler.FloorValue)
	targetInactiveProposals := math.NewIntFromUint64(params.MinInitialDepositThrottler.TargetProposals)
	k := params.MinInitialDepositThrottler.DecreaseSensitivityTargetDistance
	var alpha math.LegacyDec

	numInactiveProposals := math.NewIntFromUint64(keeper.GetInactiveProposalsNumber(ctx))
	if numInactiveProposals.GTE(targetInactiveProposals) {
		if checkElapsedTime {
			// no time-based increases
			return
		}
		alpha = math.LegacyMustNewDecFromStr(params.MinInitialDepositThrottler.IncreaseRatio)
	} else {
		distance := numInactiveProposals.Sub(targetInactiveProposals)
		if !checkElapsedTime {
			// decreases can only happen due to time-based updates
			// and if the number of active proposals is below the target
			return
		}
		alpha = math.LegacyMustNewDecFromStr(params.MinInitialDepositThrottler.DecreaseRatio)
		// ApproxRoot is here being called on a relatively small positive
		// integer (when distance < 0, ApproxRoot will return
		// `|distance|.ApproxRoot(k) * -1`) with a value of k expected to also
		// be relatively small (<= 100).
		// This is a safe operation and should not error.
		b, err := distance.ToLegacyDec().ApproxRoot(k)
		if err != nil {
			// in case of error bypass the sensitivity, i.e. assume k = 1
			b = distance.ToLegacyDec()
			logger.Error("failed to calculate ApproxRoot for min initial deposit",
				"error", err,
				"distance", distance.String(),
				"k", k,
				"fallback", "using k=1")
		}
		alpha = alpha.Mul(b)
	}
	percChange := math.LegacyOneDec().Add(alpha)
	newMinInitialDeposit := v1.GetNewMinDeposit(minInitialDepositFloor, lastMinInitialDeposit, percChange)
	keeper.SetLastMinInitialDeposit(ctx, newMinInitialDeposit, ctx.BlockTime())

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMinInitialDepositChange,
			sdk.NewAttribute(types.AttributeKeyNewMinInitialDeposit, newMinInitialDeposit.String()),
			sdk.NewAttribute(types.AttributeKeyLastMinInitialDeposit, lastMinInitialDeposit.String()),
		),
	)
}
