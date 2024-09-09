package slashing

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/slashing/keeper"
	"github.com/atomone-hub/atomone/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
)

// BeginBlocker check for infraction evidence or downtime of validators
// on every begin block
func BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	// Iterate over all the validators which *should* have signed this block
	// store whether or not they have actually signed it and slash/unbond any
	// which have missed too many blocks in a row (downtime slashing)
	for _, voteInfo := range req.LastCommitInfo.GetVotes() {
		k.HandleValidatorSignature(ctx, voteInfo.Validator.Address, voteInfo.Validator.Power, voteInfo.SignedLastBlock)
	}
}
