package keeper

import (
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
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
	activeProposalsNumber := keeper.GetActiveProposalsNumber(ctx)
	keeper.SetActiveProposalsNumber(ctx, activeProposalsNumber+1)
}

// DecrementActiveProposalsNumber decrements the number of active proposals by one
func (keeper Keeper) DecrementActiveProposalsNumber(ctx sdk.Context) {
	activeProposalsNumber := keeper.GetActiveProposalsNumber(ctx)
	keeper.SetActiveProposalsNumber(ctx, activeProposalsNumber-1)
}

// SetLastMinDeposit updates the last min deposit and last min deposit time
// used to record these values the last time the number of active proposals changed
func (keeper Keeper) SetLastMinDeposit(ctx sdk.Context, minDeposit sdk.Coin) {
	store := ctx.KVStore(keeper.storeKey)
	bz, err := keeper.cdc.Marshal(&minDeposit)
	if err != nil {
		panic(err)
	}
	store.Set(types.LastMinDepositKey, bz)

	blockTime := ctx.BlockTime()
	bz = sdk.FormatTimeBytes(blockTime)
	store.Set(types.LastMinDepositTimeKey, bz)
}

// GetLastMinDeposit returns the last min deposit and the time it was set
func (keeper Keeper) GetLastMinDeposit(ctx sdk.Context) (lastMinDeposit sdk.Coin, lastMinDepositTime time.Time) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(types.LastMinDepositKey)
	if bz == nil {
		return sdk.Coin{}, time.Time{}
	}
	err := keeper.cdc.Unmarshal(bz, &lastMinDeposit)
	if err != nil {
		panic(err)
	}

	bz = store.Get(types.LastMinDepositTimeKey)
	if bz == nil {
		return sdk.Coin{}, time.Time{}
	}
	lastMinDepositTime, err = sdk.ParseTimeBytes(bz)
	if err != nil {
		panic(err)
	}
	return lastMinDeposit, lastMinDepositTime
}

// GetCurrentMinDeposit returns the (dynamic) minimum deposit currently required for a proposal
func (keeper Keeper) GetCurrentMinDeposit(ctx sdk.Context) (sdk.Coin, error) {
	params := keeper.GetParams(ctx)
	paramMinDeposit := params.MinDeposit[0]
	tick := params.MinDepositUpdatePeriod
	targetActiveProposals := math.NewIntFromUint64(params.TargetActiveProposals)
	k := params.TargetPropsDistanceSensitivity
	a := math.LegacyZeroDec()
	b := math.ZeroInt()

	numActiveProposals := math.NewIntFromUint64(keeper.GetActiveProposalsNumber(ctx))

	if numActiveProposals.GT(targetActiveProposals) {
		a = params.DepositIncreaseRatio
	} else {
		a = params.DepositDecreaseRatio
		b = math.OneInt()
	}

	c1, err := numActiveProposals.Sub(targetActiveProposals).Sub(b).Abs().ToLegacyDec().ApproxRoot(k)
	if err != nil {
		return sdk.Coin{}, err
	}
	c := a.Mul(c1)

	lastMinDeposit, lastMinDepositTime := keeper.GetLastMinDeposit(ctx)

	// get number of ticks passed since last update
	ticksPassed := ctx.BlockTime().Sub(lastMinDepositTime).Nanoseconds() / tick.Nanoseconds()

	currMinDepositAmt := lastMinDeposit.Amount.ToLegacyDec().Mul(math.LegacyOneDec().Add(c.Power(ticksPassed))).TruncateInt()
	currMinDeposit := sdk.NewCoin(paramMinDeposit.Denom, currMinDepositAmt)
	if currMinDepositAmt.LT(paramMinDeposit.Amount) {
		currMinDeposit.Amount = paramMinDeposit.Amount
	}

	return currMinDeposit, nil
}
