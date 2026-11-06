package migrations

import (
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ClientKeeper defines the IBC client keeper interface needed to prune
// spectre consensus states.
type ClientKeeper interface {
	IterateClientStates(ctx sdk.Context, prefix []byte, cb func(string, exported.ClientState) bool)
	ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore
	Logger(ctx sdk.Context) log.Logger
}
