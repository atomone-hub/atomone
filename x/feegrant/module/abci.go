package module

import (
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/feegrant/keeper"
)

func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	k.RemoveExpiredAllowances(ctx)
}
