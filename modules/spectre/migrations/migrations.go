package migrations

import (
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/modules/spectre"
)

// PruneConsensusStates prunes the consensus states of every spectre client
// down to a window of keepSlots slots below each client's latest finalized
// slot. This function may optionally be called during in-place store
// migrations; in-flight packets must no longer reference the pruned heights.
func PruneConsensusStates(ctx sdk.Context, cdc codec.BinaryCodec, clientKeeper ClientKeeper, keepSlots uint64) (int, error) {
	var clientIDs []string
	clientKeeper.IterateClientStates(ctx, []byte(spectre.ModuleName), func(clientID string, _ exported.ClientState) bool {
		clientIDs = append(clientIDs, clientID)
		return false
	})

	// keep track of the total consensus states pruned so chains can
	// understand how much space is saved when the migration is run
	var totalPruned int

	for _, clientID := range clientIDs {
		clientStore := clientKeeper.ClientStore(ctx, clientID)

		pruned, err := spectre.PruneConsensusStates(clientStore, cdc, keepSlots)
		if err != nil {
			return 0, errorsmod.Wrapf(err, "clientID %s", clientID)
		}
		totalPruned += pruned
	}

	clientKeeper.Logger(ctx).Info("pruned spectre consensus states", "total", totalPruned)

	return totalPruned, nil
}
