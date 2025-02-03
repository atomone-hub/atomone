package gov

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/keeper"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// EndBlocker called every block, process inflation, update validator set.
func EndBlocker(ctx sdk.Context, keeper *keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	logger := keeper.Logger(ctx)

	// delete dead proposals from store and returns theirs deposits.
	// A proposal is dead when it's inactive and didn't get enough deposit on time to get into voting phase.
	keeper.IterateInactiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {
		keeper.DeleteProposal(ctx, proposal.Id)

		params := keeper.GetParams(ctx)
		if !params.BurnProposalDepositPrevote {
			keeper.RefundAndDeleteDeposits(ctx, proposal.Id) // refund deposit if proposal got removed without getting 100% of the proposal
		} else {
			keeper.DeleteAndBurnDeposits(ctx, proposal.Id) // burn the deposit if proposal got removed without getting 100% of the proposal
		}

		// called when proposal become inactive
		keeper.Hooks().AfterProposalFailedMinDeposit(ctx, proposal.Id)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeInactiveProposal,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, types.AttributeValueProposalDropped),
			),
		)

		logger.Info(
			"proposal did not meet minimum deposit; deleted",
			"proposal", proposal.Id,
			"min_deposit", sdk.NewCoins(params.MinDeposit...).String(),
			"total_deposit", sdk.NewCoins(proposal.TotalDeposit...).String(),
		)

		return false
	})

	// fetch proposals that are due to be checked for quorum
	keeper.IterateQuorumCheckQueue(ctx, ctx.BlockTime(),
		func(proposal v1.Proposal, endTime time.Time, quorumCheckEntry v1.QuorumCheckQueueEntry) bool {
			params := keeper.GetParams(ctx)
			// remove from queue
			keeper.RemoveFromQuorumCheckQueue(ctx, proposal.Id, endTime)
			// check if proposal passed quorum
			quorum, err := keeper.HasReachedQuorum(ctx, proposal)
			if err != nil {
				return false
			}
			logMsg := "proposal did not pass quorum after timeout, but was removed from quorum check queue"
			tagValue := types.AttributeValueProposalQuorumNotMet

			if quorum {
				logMsg = "proposal passed quorum before timeout, vote period was not extended"
				tagValue = types.AttributeValueProposalQuorumMet
				if quorumCheckEntry.QuorumChecksDone > 0 {
					// proposal passed quorum after timeout, extend voting period.
					// canonically, we consider the first quorum check to be "right after" the  quorum timeout has elapsed,
					// so if quorum is reached at the first check, we don't extend the voting period.
					endTime := ctx.BlockTime().Add(*params.MaxVotingPeriodExtension)
					logMsg = fmt.Sprintf("proposal passed quorum after timeout, but vote end %s is already after %s", proposal.VotingEndTime, endTime)
					if endTime.After(*proposal.VotingEndTime) {
						logMsg = fmt.Sprintf("proposal passed quorum after timeout, vote end was extended from %s to %s", proposal.VotingEndTime, endTime)
						// Update ActiveProposalsQueue with new VotingEndTime
						keeper.RemoveFromActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)
						proposal.VotingEndTime = &endTime
						keeper.InsertActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)
						keeper.SetProposal(ctx, proposal)
					}
				}
			} else if quorumCheckEntry.QuorumChecksDone < quorumCheckEntry.QuorumCheckCount && proposal.VotingEndTime.After(ctx.BlockTime()) {
				// proposal did not pass quorum and is still active, add back to queue with updated time key and counter.
				// compute time interval between quorum checks
				quorumCheckPeriod := proposal.VotingEndTime.Sub(*quorumCheckEntry.QuorumTimeoutTime)
				t := quorumCheckPeriod / time.Duration(quorumCheckEntry.QuorumCheckCount)
				// find time for next quorum check
				nextQuorumCheckTime := endTime.Add(t)
				if !nextQuorumCheckTime.After(ctx.BlockTime()) {
					// next quorum check time is in the past, so add enough time intervals to get to the next quorum check time in the future.
					d := ctx.BlockTime().Sub(nextQuorumCheckTime)
					n := d / t
					nextQuorumCheckTime = nextQuorumCheckTime.Add(t * (n + 1))
				}
				if nextQuorumCheckTime.After(*proposal.VotingEndTime) {
					// next quorum check time is after the voting period ends, so adjust it to be equal to the voting period end time
					nextQuorumCheckTime = *proposal.VotingEndTime
				}
				quorumCheckEntry.QuorumChecksDone++
				keeper.InsertQuorumCheckQueue(ctx, proposal.Id, nextQuorumCheckTime, quorumCheckEntry)

				logMsg = fmt.Sprintf("proposal did not pass quorum after timeout, next check happening at %s", nextQuorumCheckTime)
			}

			logger.Info(
				"proposal quorum check",
				"proposal", proposal.Id,
				"title", proposal.Title,
				"results", logMsg,
			)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeQuorumCheck,
					sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
					sdk.NewAttribute(types.AttributeKeyProposalResult, tagValue),
				),
			)

			return false
		})

	// fetch active proposals whose voting periods have ended (are passed the block time)
	keeper.IterateActiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {
		var tagValue, logMsg string

		passes, burnDeposits, participation, tallyResults := keeper.Tally(ctx, proposal)

		if burnDeposits {
			keeper.DeleteAndBurnDeposits(ctx, proposal.Id)
		} else {
			keeper.RefundAndDeleteDeposits(ctx, proposal.Id)
		}

		if passes {
			var (
				idx    int
				events sdk.Events
				msg    sdk.Msg
			)

			// attempt to execute all messages within the passed proposal
			// Messages may mutate state thus we use a cached context. If one of
			// the handlers fails, no state mutation is written and the error
			// message is logged.
			cacheCtx, writeCache := ctx.CacheContext()
			messages, err := proposal.GetMsgs()
			if err == nil {
				for idx, msg = range messages {
					handler := keeper.Router().Handler(msg)
					var res *sdk.Result
					res, err = safeExecuteHandler(cacheCtx, msg, handler)
					if err != nil {
						break
					}

					events = append(events, res.GetEvents()...)
				}
			}

			// `err == nil` when all handlers passed.
			// Or else, `idx` and `err` are populated with the msg index and error.
			if err == nil {
				proposal.Status = v1.StatusPassed
				tagValue = types.AttributeValueProposalPassed
				logMsg = "passed"

				// write state to the underlying multi-store
				writeCache()

				// propagate the msg events to the current context
				ctx.EventManager().EmitEvents(events)
			} else {
				proposal.Status = v1.StatusFailed
				tagValue = types.AttributeValueProposalFailed
				logMsg = fmt.Sprintf("passed, but msg %d (%s) failed on execution: %s", idx, sdk.MsgTypeURL(msg), err)
			}
		} else {
			proposal.Status = v1.StatusRejected
			tagValue = types.AttributeValueProposalRejected
			logMsg = "rejected"
		}

		proposal.FinalTallyResult = &tallyResults

		keeper.SetProposal(ctx, proposal)
		keeper.RemoveFromActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)
		keeper.UpdateParticipationEMA(ctx, participation)

		// when proposal become active
		keeper.Hooks().AfterProposalVotingPeriodEnded(ctx, proposal.Id)

		logger.Info(
			"proposal tallied",
			"proposal", proposal.Id,
			"results", logMsg,
		)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeActiveProposal,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, tagValue),
			),
		)
		return false
	})
}

// executes handle(msg) and recovers from panic.
func safeExecuteHandler(ctx sdk.Context, msg sdk.Msg, handler baseapp.MsgServiceHandler,
) (res *sdk.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handling x/gov proposal msg [%s] PANICKED: %v", msg, r)
		}
	}()
	res, err = handler(ctx, msg)
	return
}
