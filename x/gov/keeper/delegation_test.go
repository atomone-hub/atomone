package keeper_test

import (
	"testing"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestGovernanceDelegate(t *testing.T) {
	assert := assert.New(t)
	govKeeper, mocks, _, ctx := setupGovKeeper(t, mockAccountKeeperExpectations)
	s := newFixture(t, ctx, 2, 3, 2, govKeeper, mocks)
	// Setup the delegators
	s.delegate(s.delAddrs[0], s.valAddrs[0], 1)
	s.delegate(s.delAddrs[0], s.valAddrs[1], 2)
	s.delegate(s.delAddrs[1], s.valAddrs[0], 5)
	s.delegate(s.delAddrs[2], s.valAddrs[1], 8)

	// Delegate to active governor
	govKeeper.DelegateToGovernor(ctx, s.delAddrs[0], s.activeGovernors[0].GetAddress())
	govKeeper.DelegateToGovernor(ctx, s.delAddrs[1], s.activeGovernors[0].GetAddress())
	// Delegate to inactive governor
	govKeeper.DelegateToGovernor(ctx, s.delAddrs[2], s.inactiveGovernor.GetAddress())

	// Assert GetGovernanceDelegation
	deleg1, found := govKeeper.GetGovernanceDelegation(ctx, s.delAddrs[0])
	if assert.True(found, "deleg1 not found") {
		assert.Equal(deleg1.DelegatorAddress, s.delAddrs[0].String())
		assert.Equal(deleg1.GovernorAddress, s.activeGovernors[0].GovernorAddress)
	}
	deleg2, found := govKeeper.GetGovernanceDelegation(ctx, s.delAddrs[1])
	if assert.True(found, "deleg2 not found") {
		assert.Equal(deleg2.DelegatorAddress, s.delAddrs[1].String())
		assert.Equal(deleg2.GovernorAddress, s.activeGovernors[0].GovernorAddress)
	}
	deleg3, found := govKeeper.GetGovernanceDelegation(ctx, s.delAddrs[2])
	if assert.True(found, "deleg3 not found") {
		assert.Equal(deleg3.DelegatorAddress, s.delAddrs[2].String())
		assert.Equal(deleg3.GovernorAddress, s.inactiveGovernor.GovernorAddress)
	}

	// Assert IterateGovernorDelegations
	var delegs []v1.GovernanceDelegation
	govKeeper.IterateGovernorDelegations(ctx, s.activeGovernors[0].GetAddress(),
		func(i int64, d v1.GovernanceDelegation) bool {
			delegs = append(delegs, d)
			return false
		})
	assert.ElementsMatch(delegs, []v1.GovernanceDelegation{deleg1, deleg2})
	delegs = nil
	govKeeper.IterateGovernorDelegations(ctx, s.inactiveGovernor.GetAddress(),
		func(i int64, d v1.GovernanceDelegation) bool {
			delegs = append(delegs, d)
			return false
		})
	assert.ElementsMatch(delegs, []v1.GovernanceDelegation{deleg3})

	// Assert GetAllGovernanceDelegationsByGovernor
	allDelegs := govKeeper.GetAllGovernanceDelegationsByGovernor(ctx, s.activeGovernors[0].GetAddress())
	assert.ElementsMatch(allDelegs, []*v1.GovernanceDelegation{&deleg1, &deleg2})
	allDelegs = govKeeper.GetAllGovernanceDelegationsByGovernor(ctx, s.inactiveGovernor.GetAddress())
	assert.ElementsMatch(allDelegs, []*v1.GovernanceDelegation{&deleg3})

	// Assert GetGovernorValShares
	valShare1, found := govKeeper.GetGovernorValShares(ctx, s.activeGovernors[0].GetAddress(), s.valAddrs[0])
	if assert.True(found, "valShare1 not found") {
		assert.Equal(valShare1, v1.GovernorValShares{
			GovernorAddress:  s.activeGovernors[0].GovernorAddress,
			ValidatorAddress: s.valAddrs[0].String(),
			Shares:           sdk.NewDec(1 + 5),
		})
	}
	valShare2, found := govKeeper.GetGovernorValShares(ctx, s.activeGovernors[0].GetAddress(), s.valAddrs[1])
	if assert.True(found, "valShare2 not found") {
		assert.Equal(valShare2, v1.GovernorValShares{
			GovernorAddress:  s.activeGovernors[0].GovernorAddress,
			ValidatorAddress: s.valAddrs[1].String(),
			Shares:           sdk.NewDec(2),
		})
	}
	_, found = govKeeper.GetGovernorValShares(ctx, s.inactiveGovernor.GetAddress(), s.valAddrs[0])
	assert.False(found)
	valShare3, found := govKeeper.GetGovernorValShares(ctx, s.inactiveGovernor.GetAddress(), s.valAddrs[1])
	if assert.True(found, "valShare3 not found") {
		assert.Equal(valShare3, v1.GovernorValShares{
			GovernorAddress:  s.inactiveGovernor.GovernorAddress,
			ValidatorAddress: s.valAddrs[1].String(),
			Shares:           sdk.NewDec(8),
		})
	}

	// Assert GetAllGovernorValShares
	activeGovValShares := govKeeper.GetAllGovernorValShares(ctx, s.activeGovernors[0].GetAddress())
	assert.ElementsMatch(activeGovValShares, []v1.GovernorValShares{valShare1, valShare2})
	inactiveGovValShares := govKeeper.GetAllGovernorValShares(ctx, s.inactiveGovernor.GetAddress())
	assert.ElementsMatch(inactiveGovValShares, []v1.GovernorValShares{valShare3})

	// Assert IterateGovernorValShares
	var valShares []v1.GovernorValShares
	govKeeper.IterateGovernorValShares(ctx, s.activeGovernors[0].GetAddress(), func(i int64, v v1.GovernorValShares) bool {
		valShares = append(valShares, v)
		return false
	})
	assert.ElementsMatch(valShares, activeGovValShares)
	valShares = nil
	govKeeper.IterateGovernorValShares(ctx, s.inactiveGovernor.GetAddress(), func(i int64, v v1.GovernorValShares) bool {
		valShares = append(valShares, v)
		return false
	})
	assert.ElementsMatch(valShares, inactiveGovValShares)

	// Assert RedelegateToGovernor
	govKeeper.RedelegateToGovernor(ctx, s.delAddrs[0], s.inactiveGovernor.GetAddress())
	allDelegs = govKeeper.GetAllGovernanceDelegationsByGovernor(ctx, s.activeGovernors[0].GetAddress())
	assert.ElementsMatch(allDelegs, []*v1.GovernanceDelegation{&deleg2})
	allDelegs = govKeeper.GetAllGovernanceDelegationsByGovernor(ctx, s.inactiveGovernor.GetAddress())
	deleg1.GovernorAddress = s.inactiveGovernor.GovernorAddress
	assert.ElementsMatch(allDelegs, []*v1.GovernanceDelegation{&deleg1, &deleg3})
	valShare1, found = govKeeper.GetGovernorValShares(ctx, s.activeGovernors[0].GetAddress(), s.valAddrs[0])
	if assert.True(found, "valShare1 not found") {
		assert.Equal(valShare1, v1.GovernorValShares{
			GovernorAddress:  s.activeGovernors[0].GovernorAddress,
			ValidatorAddress: s.valAddrs[0].String(),
			Shares:           sdk.NewDec(5),
		})
	}
	_, found = govKeeper.GetGovernorValShares(ctx, s.activeGovernors[0].GetAddress(), s.valAddrs[1])
	assert.False(found)
	valShare2, found = govKeeper.GetGovernorValShares(ctx, s.inactiveGovernor.GetAddress(), s.valAddrs[0])
	if assert.True(found, "valShare2 not found") {
		assert.Equal(valShare2, v1.GovernorValShares{
			GovernorAddress:  s.inactiveGovernor.GovernorAddress,
			ValidatorAddress: s.valAddrs[0].String(),
			Shares:           sdk.NewDec(1),
		})
	}
	valShare3, found = govKeeper.GetGovernorValShares(ctx, s.inactiveGovernor.GetAddress(), s.valAddrs[1])
	if assert.True(found, "valShare3 not found") {
		assert.Equal(valShare3, v1.GovernorValShares{
			GovernorAddress:  s.inactiveGovernor.GovernorAddress,
			ValidatorAddress: s.valAddrs[1].String(),
			Shares:           sdk.NewDec(10),
		})
	}

	// Assert UndelegateFromGovernor
	govKeeper.UndelegateFromGovernor(ctx, s.delAddrs[0])
	allDelegs = govKeeper.GetAllGovernanceDelegationsByGovernor(ctx, s.activeGovernors[0].GetAddress())
	assert.ElementsMatch(allDelegs, []*v1.GovernanceDelegation{&deleg2})
	allDelegs = govKeeper.GetAllGovernanceDelegationsByGovernor(ctx, s.inactiveGovernor.GetAddress())
	deleg1.GovernorAddress = s.inactiveGovernor.GovernorAddress
	assert.ElementsMatch(allDelegs, []*v1.GovernanceDelegation{&deleg3})
	valShare1, found = govKeeper.GetGovernorValShares(ctx, s.activeGovernors[0].GetAddress(), s.valAddrs[0])
	if assert.True(found, "valShare1 not found") {
		assert.Equal(valShare1, v1.GovernorValShares{
			GovernorAddress:  s.activeGovernors[0].GovernorAddress,
			ValidatorAddress: s.valAddrs[0].String(),
			Shares:           sdk.NewDec(5),
		})
	}
	_, found = govKeeper.GetGovernorValShares(ctx, s.activeGovernors[0].GetAddress(), s.valAddrs[1])
	assert.False(found)
	_, found = govKeeper.GetGovernorValShares(ctx, s.inactiveGovernor.GetAddress(), s.valAddrs[0])
	assert.False(found)
	valShare3, found = govKeeper.GetGovernorValShares(ctx, s.inactiveGovernor.GetAddress(), s.valAddrs[1])
	if assert.True(found, "valShare3 not found") {
		assert.Equal(valShare3, v1.GovernorValShares{
			GovernorAddress:  s.inactiveGovernor.GovernorAddress,
			ValidatorAddress: s.valAddrs[1].String(),
			Shares:           sdk.NewDec(8),
		})
	}
}
