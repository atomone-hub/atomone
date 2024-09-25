package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
)

func (keeper Keeper) GetConstitution(ctx sdk.Context) (constitution string) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(types.KeyConstitution)

	return string(bz)
}

func (keeper Keeper) SetConstitution(ctx sdk.Context, constitution string) {
	store := ctx.KVStore(keeper.storeKey)
	store.Set(types.KeyConstitution, []byte(constitution))
}

// ApplyConstitutionAmendment applies the amendment as a patch against the current constitution
// and returns the updated constitution. If the amendment cannot be applied cleanly, an error is returned.
func (k Keeper) ApplyConstitutionAmendment(ctx sdk.Context, amendment string) (updatedConstitution string, err error) {
	if amendment == "" {
		return "", types.ErrInvalidConstitutionAmendment.Wrap("amendment cannot be empty")
	}

	currentConstitution := k.GetConstitution(ctx)
	updatedConstitution, err = types.ApplyUnifiedDiff(currentConstitution, amendment)
	if err != nil {
		return "", types.ErrInvalidConstitutionAmendment.Wrapf("failed to apply amendment: %v", err)
	}

	return updatedConstitution, nil
}
