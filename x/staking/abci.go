package staking

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/staking/keeper"
	"github.com/atomone-hub/atomone/x/staking/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func BeginBlocker(ctx sdk.Context, k *keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	k.TrackHistoricalInfo(ctx)
}

// Called every block, update validator set
func EndBlocker(ctx sdk.Context, k *keeper.Keeper) []abci.ValidatorUpdate {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	return k.BlockValidatorUpdates(ctx)
}
