package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	"github.com/atomone-hub/atomone/x/coredaos/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	authority    string

	govKeeper     *govkeeper.Keeper
	stakingKeeper types.StakingKeeper

	Schema collections.Schema
	Params collections.Item[types.Params]
	// VetoCleanupQueue holds the ids of proposals vetoed during the current
	// block whose deposits and votes still need to be cleaned up, mapped to
	// their BurnDeposit flag. It is drained in the EndBlocker (see abci.go).
	VetoCleanupQueue collections.Map[uint64, bool]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	authority string,
	govKeeper *govkeeper.Keeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(err)
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		govKeeper:     govKeeper,
		stakingKeeper: stakingKeeper,
		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		VetoCleanupQueue: collections.NewMap(
			sb, types.VetoCleanupQueueKey, "veto_cleanup_queue",
			collections.Uint64Key, collections.BoolValue,
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// Logger returns a coredaos module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the address that is capable of executing a MsgUpdateParams message.
func (k Keeper) GetAuthority() string {
	return k.authority
}
