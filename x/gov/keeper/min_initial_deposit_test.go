package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/keeper"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestInactiveProposalNumber(t *testing.T) {
	assert := assert.New(t)
	k, _, _, ctx := setupGovKeeper(t)

	assert.EqualValues(0, k.GetInactiveProposalsNumber(ctx))

	k.IncrementInactiveProposalsNumber(ctx)
	k.IncrementInactiveProposalsNumber(ctx)
	assert.EqualValues(2, k.GetInactiveProposalsNumber(ctx))

	k.DecrementInactiveProposalsNumber(ctx)
	assert.EqualValues(1, k.GetInactiveProposalsNumber(ctx))

	k.SetInactiveProposalsNumber(ctx, 42)
	assert.EqualValues(42, k.GetInactiveProposalsNumber(ctx))
}

func TestGetMinInitialDeposit(t *testing.T) {
	var (
		minInitialDepositFloor   = v1.DefaultMinInitialDepositFloor
		minInitialDepositFloorX2 = minInitialDepositFloor.MulInt(sdk.NewInt(2))
		updatePeriod             = v1.DefaultMinInitialDepositUpdatePeriod
		N                        = v1.DefaultTargetProposalsInDepositPeriod

		minInitialDepositTimeFromTicks = func(ticks int) time.Time {
			return time.Now().Add(-updatePeriod*time.Duration(ticks) - time.Minute)
		}
	)
	tests := []struct {
		name                      string
		setup                     func(sdk.Context, *keeper.Keeper)
		expectedMinInitialDeposit string
	}{
		{
			name:                      "initial case no setup : expectedMinInitialDeposit=minInitialDepositFloor",
			expectedMinInitialDeposit: minInitialDepositFloor.String(),
		},
		{
			name: "n=N-1 lastMinInitialDeposit=minInitialDepositFloor ticksPassed=0 : expectedMinInitialDeposit=minInitialDepositFloor",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, 0)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloor, minInitialDepositTimeFromTicks(0))
			},
			expectedMinInitialDeposit: minInitialDepositFloor.String(),
		},
		{
			name: "n=N lastMinInitialDeposit=minInitialDepositFloor ticksPassed=0 : expectedMinInitialDeposit=minInitialDepositFloor",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloor, minInitialDepositTimeFromTicks(0))
			},
			expectedMinInitialDeposit: minInitialDepositFloor.String(),
		},
		{
			name: "n=N+1 lastMinInitialDeposit=minInitialDepositFloor ticksPassed=0 : expectedMinInitialDeposit>minInitialDepositFloor",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N+1)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloor, minInitialDepositTimeFromTicks(0))
			},
			expectedMinInitialDeposit: "101000stake",
		},
		{
			name: "n=N+1 lastMinInitialDeposit=otherCoins ticksPassed=0 : expectedMinInitialDeposit>minInitialDepositFloor",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N+1)
				k.SetLastMinInitialDeposit(ctx, sdk.NewCoins(
					sdk.NewInt64Coin("xxx", 1_000_000_000),
				),
					minInitialDepositTimeFromTicks(0),
				)
			},
			expectedMinInitialDeposit: "101000stake",
		},
		{
			name: "n=N-1 lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=0 : minInitialDeposit<lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N-1)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(0))
			},
			expectedMinInitialDeposit: "199000stake",
		},
		{
			name: "n=N lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=0 : expectedMinInitialDeposit=lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(0))
			},
			expectedMinInitialDeposit: minInitialDepositFloorX2.String(),
		},
		{
			name: "n=N+1 lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=0 : expectedMinInitialDeposit>lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N+1)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(0))
			},
			expectedMinInitialDeposit: "202000stake",
		},
		{
			name: "n=N-1 lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=1 : expectedMinInitialDeposit<lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N-1)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(1))
			},
			expectedMinInitialDeposit: "199000stake",
		},
		{
			name: "n=N lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=1 : expectedMinInitialDeposit=lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(1))
			},
			expectedMinInitialDeposit: minInitialDepositFloorX2.String(),
		},
		{
			name: "n=N+1 lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=1 : expectedMinInitialDeposit>lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N+1)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(1))
			},
			expectedMinInitialDeposit: "202000stake",
		},
		{
			name: "n=N-1 lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=2 : expectedMinInitialDeposit<lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N-1)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(2))
			},
			expectedMinInitialDeposit: "198005stake",
		},
		{
			name: "n=N lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=2 : expectedMinInitialDeposit=lastMinInitialDeposit*2",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(2))
			},
			expectedMinInitialDeposit: minInitialDepositFloorX2.String(),
		},
		{
			name: "n=N+1 lastMinInitialDeposit=minInitialDepositFloor*2 ticksPassed=2 : expectedMinInitialDeposit=lastMinInitialDeposit",
			setup: func(ctx sdk.Context, k *keeper.Keeper) {
				k.SetInactiveProposalsNumber(ctx, N+1)
				k.SetLastMinInitialDeposit(ctx, minInitialDepositFloorX2, minInitialDepositTimeFromTicks(2))
			},
			expectedMinInitialDeposit: "204020stake",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			k, _, _, ctx := setupGovKeeper(t)
			if tt.setup != nil {
				tt.setup(ctx, k)
			}
			minInitialDeposit := k.GetMinInitialDeposit(ctx)
			assert.Equal(tt.expectedMinInitialDeposit, minInitialDeposit.String())
		})
	}
}
