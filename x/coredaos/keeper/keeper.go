package keeper

import (
	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/coredaos/types"
)

type Keeper struct {
	cdc       codec.BinaryCodec
	storeKey  storetypes.StoreKey
	authority string

	govKeeper     types.GovKeeper
	stakingKeeper types.StakingKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority string,
	govKeeper types.GovKeeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(err)
	}

	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		authority:     authority,
		govKeeper:     govKeeper,
		stakingKeeper: stakingKeeper,
	}
}

// Logger returns a coredaos module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the address that is capable of executing a MsgUpdateParams message.
func (k Keeper) GetAuthority() string {
	return k.authority
}
