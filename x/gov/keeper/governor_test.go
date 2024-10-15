package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestGovernor(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	govKeeper, _, _, ctx := setupGovKeeper(t)
	addrs := simtestutil.CreateRandomAccounts(3)
	govAddrs := convertAddrsToGovAddrs(addrs)

	// Add 2 governors
	gov1Desc := v1.NewGovernorDescription("moniker1", "id1", "website1", "sec1", "detail1")
	gov1, err := v1.NewGovernor(govAddrs[0].String(), gov1Desc, time.Now().UTC())
	require.NoError(err)
	gov2Desc := v1.NewGovernorDescription("moniker2", "id2", "website2", "sec2", "detail2")
	gov2, err := v1.NewGovernor(govAddrs[1].String(), gov2Desc, time.Now().UTC())
	gov2.Status = v1.Inactive
	require.NoError(err)
	govKeeper.SetGovernor(ctx, gov1)
	govKeeper.SetGovernor(ctx, gov2)

	// Get gov1
	gov, found := govKeeper.GetGovernor(ctx, govAddrs[0])
	if assert.True(found, "cant find gov1") {
		assert.Equal(gov1, gov)
	}

	// Get gov2
	gov, found = govKeeper.GetGovernor(ctx, govAddrs[1])
	if assert.True(found, "cant find gov2") {
		assert.Equal(gov2, gov)
	}

	// Get all govs
	govs := govKeeper.GetAllGovernors(ctx)
	if assert.Len(govs, 2, "expected 2 governors") {
		// Insert order is not preserved, order is related to the address which is
		// generated randomly, so the order of govs is random.
		for i := 0; i < 2; i++ {
			switch govs[i].GetAddress().String() {
			case gov1.GetAddress().String():
				assert.Equal(gov1, *govs[i])
			case gov2.GetAddress().String():
				assert.Equal(gov2, *govs[i])
			}
		}
	}

	// Get all active govs
	govs = govKeeper.GetAllActiveGovernors(ctx)
	if assert.Len(govs, 1, "expected 1 active governor") {
		assert.Equal(gov1, *govs[0])
	}

	// IterateGovernors
	govs = nil
	govKeeper.IterateGovernors(ctx, func(i int64, govI v1.GovernorI) bool {
		gov := govI.(v1.Governor)
		govs = append(govs, &gov)
		return false
	})
	if assert.Len(govs, 2, "expected 2 governors") {
		for i := 0; i < 2; i++ {
			switch govs[i].GetAddress().String() {
			case gov1.GetAddress().String():
				assert.Equal(gov1, *govs[i])
			case gov2.GetAddress().String():
				assert.Equal(gov2, *govs[i])
			}
		}
	}
}

func TestValidateGovernorMinSelfDelegation(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*fixture) v1.Governor
		selfDelegation bool
		valDelegations []stakingtypes.Delegation
		expectedPanic  bool
		expectedValid  bool
	}{
		{
			name: "inactive governor",
			setup: func(s *fixture) v1.Governor {
				return s.inactiveGovernor
			},
			expectedPanic: false,
			expectedValid: false,
		},
		{
			name: "active governor w/o self delegation w/o validator delegation",
			setup: func(s *fixture) v1.Governor {
				return s.activeGovernors[0]
			},
			expectedPanic: true,
			expectedValid: false,
		},
		{
			name: "active governor w self delegation w/o validator delegation",
			setup: func(s *fixture) v1.Governor {
				govAddr := s.activeGovernors[0].GetAddress()
				delAddr := sdk.AccAddress(govAddr)
				s.keeper.DelegateToGovernor(s.ctx, delAddr, govAddr)
				return s.activeGovernors[0]
			},
			expectedPanic: false,
			expectedValid: false,
		},
		{
			name: "active governor w self delegation w not enough validator delegation",
			setup: func(s *fixture) v1.Governor {
				govAddr := s.activeGovernors[0].GetAddress()
				delAddr := sdk.AccAddress(govAddr)
				s.keeper.DelegateToGovernor(s.ctx, delAddr, govAddr)
				s.delegate(delAddr, s.valAddrs[0], 1)
				return s.activeGovernors[0]
			},
			expectedPanic: false,
			expectedValid: false,
		},
		{
			name: "active governor w self delegation w enough validator delegation",
			setup: func(s *fixture) v1.Governor {
				govAddr := s.activeGovernors[0].GetAddress()
				delAddr := sdk.AccAddress(govAddr)
				s.keeper.DelegateToGovernor(s.ctx, delAddr, govAddr)
				s.delegate(delAddr, s.valAddrs[0], v1.DefaultMinGovernorSelfDelegation.Int64())
				return s.activeGovernors[0]
			},
			expectedPanic: false,
			expectedValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			govKeeper, mocks, _, ctx := setupGovKeeper(t, mockAccountKeeperExpectations)
			s := newFixture(t, ctx, 2, 2, 2, govKeeper, mocks)
			governor := tt.setup(s)

			if tt.expectedPanic {
				assert.Panics(t, func() { govKeeper.ValidateGovernorMinSelfDelegation(ctx, governor) })
			} else {
				valid := govKeeper.ValidateGovernorMinSelfDelegation(ctx, governor)

				assert.Equal(t, tt.expectedValid, valid, "return of ValidateGovernorMinSelfDelegation")
			}
		})
	}
}
