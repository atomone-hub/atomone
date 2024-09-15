package gov

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/keeper"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// InitGenesis - store genesis parameters
func InitGenesis(ctx sdk.Context, ak types.AccountKeeper, bk types.BankKeeper, k *keeper.Keeper, data *v1.GenesisState) {
	k.SetProposalID(ctx, data.StartingProposalId)
	if err := k.SetParams(ctx, *data.Params); err != nil {
		panic(fmt.Sprintf("%s module params has not been set", types.ModuleName))
	}

	// check if the deposits pool account exists
	moduleAcc := k.GetGovernanceAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	var totalDeposits sdk.Coins
	for _, deposit := range data.Deposits {
		k.SetDeposit(ctx, *deposit)
		totalDeposits = totalDeposits.Add(deposit.Amount...)
	}

	for _, vote := range data.Votes {
		k.SetVote(ctx, *vote)
	}

	for _, proposal := range data.Proposals {
		switch proposal.Status {
		case v1.StatusDepositPeriod:
			k.InsertInactiveProposalQueue(ctx, proposal.Id, *proposal.DepositEndTime)
		case v1.StatusVotingPeriod:
			k.InsertActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)
		}
		k.SetProposal(ctx, *proposal)
	}

	// if account has zero balance it probably means it's not set, so we set it
	balance := bk.GetAllBalances(ctx, moduleAcc.GetAddress())
	if balance.IsZero() {
		ak.SetModuleAccount(ctx, moduleAcc)
	}

	// check if total deposits equals balance, if it doesn't panic because there were export/import errors
	if !balance.IsEqual(totalDeposits) {
		panic(fmt.Sprintf("expected module account was %s but we got %s", balance.String(), totalDeposits.String()))
	}

	// set governors
	for _, governor := range data.Governors {
		// check that base account exists
		accAddr := sdk.AccAddress(governor.GetAddress())
		acc := ak.GetAccount(ctx, accAddr)
		if acc == nil {
			panic(fmt.Sprintf("account %s does not exist", accAddr.String()))
		}

		k.SetGovernor(ctx, *governor)
		if governor.IsActive() {
			k.DelegateToGovernor(ctx, accAddr, governor.GetAddress())
		}
	}
	// set governance delegations
	for _, delegation := range data.GovernanceDelegations {
		delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)
		govAddr := types.MustGovernorAddressFromBech32(delegation.GovernorAddress)
		// check delegator exists
		acc := ak.GetAccount(ctx, delAddr)
		if acc == nil {
			panic(fmt.Sprintf("account %s does not exist", delAddr.String()))
		}
		// check governor exists
		_, found := k.GetGovernor(ctx, govAddr)
		if !found {
			panic(fmt.Sprintf("governor %s does not exist", govAddr.String()))
		}

		// if account is active governor and delegation is not to self, error
		delGovAddr := types.GovernorAddress(delAddr)
		if _, found = k.GetGovernor(ctx, delGovAddr); found && !delGovAddr.Equals(govAddr) {
			panic(fmt.Sprintf("account %s is an active governor and cannot delegate", delAddr.String()))
		}

		k.DelegateToGovernor(ctx, delAddr, govAddr)
	}
}

// ExportGenesis - output genesis parameters
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *v1.GenesisState {
	startingProposalID, _ := k.GetProposalID(ctx)
	proposals := k.GetProposals(ctx)
	params := k.GetParams(ctx)
	governors := k.GetAllGovernors(ctx)

	var proposalsDeposits v1.Deposits
	var proposalsVotes v1.Votes
	for _, proposal := range proposals {
		deposits := k.GetDeposits(ctx, proposal.Id)
		proposalsDeposits = append(proposalsDeposits, deposits...)

		votes := k.GetVotes(ctx, proposal.Id)
		proposalsVotes = append(proposalsVotes, votes...)
	}

	var governanceDelegations []*v1.GovernanceDelegation
	for _, g := range governors {
		delegations := k.GetAllGovernanceDelegationsByGovernor(ctx, g.GetAddress())
		governanceDelegations = append(governanceDelegations, delegations...)
	}
	return &v1.GenesisState{
		StartingProposalId:    startingProposalID,
		Deposits:              proposalsDeposits,
		Votes:                 proposalsVotes,
		Proposals:             proposals,
		Params:                &params,
		Governors:             governors,
		GovernanceDelegations: governanceDelegations,
	}
}
