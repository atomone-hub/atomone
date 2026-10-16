package keeper

import (
	"context"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker performs the deferred cleanup of proposals vetoed during the block.
//
// VetoProposal only marks a proposal as vetoed and enqueues its id; the
// potentially unbounded work — refunding or burning every deposit and deleting
// every vote — is carried out here. EndBlock runs under the infinite block gas
// meter, so this cleanup cannot be priced out of a block the way it could if it
// ran inside the metered veto transaction. This mirrors how x/gov cleans up
// rejected proposals from its own EndBlocker.
//
// This module's EndBlocker MUST run before x/gov's (see orderEndBlockers): the
// cleanup deletes the vetoed proposal's votes, and gov's quorum check must not
// observe those votes, otherwise it could pass quorum and re-insert the vetoed
// proposal into the ActiveProposalsQueue.
//
// The queue is drained fully every block, so it is always empty at a block
// boundary and needs no genesis import/export.
func (k Keeper) EndBlocker(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return k.VetoCleanupQueue.Walk(ctx, nil, func(proposalID uint64, burnDeposit bool) (bool, error) {
		// Each proposal is cleaned up in its own cached context: this is deferred
		// best-effort work, so a failure must never halt the chain. On success we
		// commit the changes (which include removing the proposal from the queue);
		// on error we log and leave the entry queued to be retried next block.
		cacheCtx, writeCache := ctx.CacheContext()
		if err := k.cleanupVetoedProposal(cacheCtx, proposalID, burnDeposit); err != nil {
			k.Logger(ctx).Error(
				"failed to clean up vetoed proposal deposits/votes, will retry next block",
				"proposal_id", proposalID,
				"error", err,
			)
			return false, nil
		}
		writeCache()

		return false, nil
	})
}

// cleanupVetoedProposal refunds or burns every deposit of the proposal, deletes
// every vote, and removes the proposal from the veto cleanup queue. It follows
// the same logic as x/gov/abci.go for rejected proposals. The collections walk
// in EndBlocker tolerates the queue removal happening within the iteration.
func (k Keeper) cleanupVetoedProposal(ctx sdk.Context, proposalID uint64, burnDeposit bool) error {
	if burnDeposit {
		if err := k.govKeeper.DeleteAndBurnDeposits(ctx, proposalID); err != nil {
			return err
		}
	} else {
		if err := k.govKeeper.RefundAndDeleteDeposits(ctx, proposalID); err != nil {
			return err
		}
	}
	// Delete all votes for the proposal. Votes are stored as
	// collections.Map[collections.Pair[uint64, sdk.AccAddress], v1.Vote].
	if err := k.govKeeper.Votes.Clear(ctx, collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposalID)); err != nil {
		return err
	}

	return k.VetoCleanupQueue.Remove(ctx, proposalID)
}
