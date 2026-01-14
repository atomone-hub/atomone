package migrations

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcgno "github.com/atomone-hub/atomone/modules/10-gno"
)

// PruneExpiredConsensusStates prunes all expired GNO consensus states. This function
// may optionally be called during in-place store migrations. The ibc store key must be provided.
func PruneExpiredConsensusStates(ctx sdk.Context, cdc codec.BinaryCodec, clientKeeper ClientKeeper) (int, error) {
	var clientIDs []string
	clientKeeper.IterateClientStates(ctx, []byte(ibcgno.Gno), func(clientID string, _ exported.ClientState) bool {
		clientIDs = append(clientIDs, clientID)
		return false
	})

	// keep track of the total consensus states pruned so chains can
	// understand how much space is saved when the migration is run
	var totalPruned int

	for _, clientID := range clientIDs {
		clientStore := clientKeeper.ClientStore(ctx, clientID)

		clientState, ok := clientKeeper.GetClientState(ctx, clientID)
		if !ok {
			return 0, errorsmod.Wrapf(clienttypes.ErrClientNotFound, "clientID %s", clientID)
		}

		gnoClientState, ok := clientState.(*ibcgno.ClientState)
		if !ok {
			return 0, errorsmod.Wrap(clienttypes.ErrInvalidClient, "client state is not GNO even though client id contains 10-gno")
		}

		totalPruned += ibcgno.PruneAllExpiredConsensusStates(ctx, clientStore, cdc, gnoClientState)
	}

	clientKeeper.Logger(ctx).Info("pruned expired gno consensus states", "total", totalPruned)

	return totalPruned, nil
}
