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
	k.SetConstitution(ctx, data.Constitution)

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

		if data.Params.QuorumCheckCount > 0 && proposal.Status == v1.StatusVotingPeriod {
			quorumTimeoutTime := proposal.VotingStartTime.Add(*data.Params.QuorumTimeout)
			quorumCheckEntry := v1.NewQuorumCheckQueueEntry(quorumTimeoutTime, data.Params.QuorumCheckCount)
			quorum := false
			if ctx.BlockTime().After(quorumTimeoutTime) {
				var err error
				quorum, err = k.HasReachedQuorum(ctx, *proposal)
				if err != nil {
					panic(err)
				}
				if !quorum {
					// since we don't export the state of the quorum check queue, we can't know how many checks were actually
					// done. However, in order to trigger a vote time extension, it is enough to have QuorumChecksDone > 0 to
					// trigger a vote time extension, so we set it to 1
					quorumCheckEntry.QuorumChecksDone = 1
				}
			}
			if !quorum {
				k.InsertQuorumCheckQueue(ctx, proposal.Id, quorumTimeoutTime, quorumCheckEntry)
			}
		}

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
}

// ExportGenesis - output genesis parameters
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *v1.GenesisState {
	startingProposalID, _ := k.GetProposalID(ctx)
	proposals := k.GetProposals(ctx)
	params := k.GetParams(ctx)
	constitution := k.GetConstitution(ctx)

	var proposalsDeposits v1.Deposits
	var proposalsVotes v1.Votes
	for _, proposal := range proposals {
		deposits := k.GetDeposits(ctx, proposal.Id)
		proposalsDeposits = append(proposalsDeposits, deposits...)

		votes := k.GetVotes(ctx, proposal.Id)
		proposalsVotes = append(proposalsVotes, votes...)
	}

	return &v1.GenesisState{
		StartingProposalId: startingProposalID,
		Deposits:           proposalsDeposits,
		Votes:              proposalsVotes,
		Proposals:          proposals,
		Params:             &params,
		Constitution:       constitution,
	}
}
