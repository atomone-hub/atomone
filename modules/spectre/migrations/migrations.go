package migrations

import (
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/modules/spectre"
)

// PruneConsensusStates prunes the consensus states of every spectre client
// down to the module pruning window below each client's latest finalized
// slot. It deletes at most maxPerClient states per client per call, so a
// single invocation does bounded work; callers repeat until it returns zero.
//
// The window equals the one membership verification accepts, so heights
// below it are rejected by verification regardless: pruning at the window
// cannot strand in-flight packets. This function may optionally be called
// during in-place store migrations.
func PruneConsensusStates(ctx sdk.Context, cdc codec.BinaryCodec, clientKeeper ClientKeeper, maxPerClient int) (int, error) {
	type client struct {
		id    string
		state *spectre.ClientState
	}
	var clients []client
	clientKeeper.IterateClientStates(ctx, []byte(spectre.ModuleName), func(clientID string, state exported.ClientState) bool {
		if cs, ok := state.(*spectre.ClientState); ok {
			clients = append(clients, client{id: clientID, state: cs})
		}
		return false
	})

	// keep track of the total consensus states pruned so chains can
	// understand how much space is saved when the migration is run
	var totalPruned int

	for _, c := range clients {
		clientStore := clientKeeper.ClientStore(ctx, c.id)

		pruned, err := spectre.PruneConsensusStates(clientStore, cdc, c.state.PruneKeepSlots(), maxPerClient)
		if err != nil {
			return 0, errorsmod.Wrapf(err, "clientID %s", c.id)
		}
		totalPruned += pruned
	}

	clientKeeper.Logger(ctx).Info("pruned spectre consensus states", "total", totalPruned)

	return totalPruned, nil
}
