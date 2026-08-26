package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

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
func (k Keeper) EndBlocker(ctx context.Context) error {
	return k.VetoCleanupQueue.Walk(ctx, nil, func(proposalID uint64, burnDeposit bool) (bool, error) {
		// follows the same logic as in x/gov/abci.go for rejected proposals
		if burnDeposit {
			if err := k.govKeeper.DeleteAndBurnDeposits(ctx, proposalID); err != nil {
				return false, errors.Wrapf(err, "error deleting and burning deposits for vetoed proposal %d", proposalID)
			}
		} else {
			if err := k.govKeeper.RefundAndDeleteDeposits(ctx, proposalID); err != nil {
				return false, errors.Wrapf(err, "error refunding and deleting deposits for vetoed proposal %d", proposalID)
			}
		}
		// Delete all votes for the proposal. Votes are stored as
		// collections.Map[collections.Pair[uint64, sdk.AccAddress], v1.Vote].
		if err := k.govKeeper.Votes.Clear(ctx, collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposalID)); err != nil {
			return false, errors.Wrapf(err, "error deleting votes for vetoed proposal %d", proposalID)
		}
		// The collections walk tolerates removing the current key within the
		// iteration, as x/gov's own EndBlocker does for its queues.
		if err := k.VetoCleanupQueue.Remove(ctx, proposalID); err != nil {
			return false, errors.Wrapf(err, "error removing vetoed proposal %d from cleanup queue", proposalID)
		}

		return false, nil
	})
}
