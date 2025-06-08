package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
)

// GetParticipationEMA returns the governance participation EMA
func (k Keeper) GetParticipationEMA(ctx sdk.Context) (participationEma math.LegacyDec) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyParticipationEMA)

	if bz == nil {
		return math.LegacyDec{}
	}

	participationEma = math.LegacyMustNewDecFromStr(string(bz))
	return participationEma
}

// SetParticipationEMA sets the governance participation EMA
func (k Keeper) SetParticipationEMA(ctx sdk.Context, participationEma math.LegacyDec) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyParticipationEMA, []byte(participationEma.String()))
}

// UpdateParticipationEMA updates the governance participation EMA
func (k Keeper) UpdateParticipationEMA(ctx sdk.Context, participation math.LegacyDec) {
	old_participationEma := k.GetParticipationEMA(ctx)
	// new_participationEma = 0.8 * old_participationEma + 0.2 * participation
	new_participationEma := old_participationEma.Mul(sdk.NewDecWithPrec(8, 1)).Add(participation.Mul(sdk.NewDecWithPrec(2, 1)))
	k.SetParticipationEMA(ctx, new_participationEma)
}

// GetQuorum returns the dynamic quorum for governance proposals calculated
// based on the participation EMA
func (k Keeper) GetQuorum(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	minQuorum := math.LegacyMustNewDecFromStr(params.MinQuorum)
	maxQuorum := math.LegacyMustNewDecFromStr(params.MaxQuorum)
	return k.getQuorum(ctx, minQuorum, maxQuorum)
}

// GetConstitutionAmendmentQuorum returns the dynamic quorum for constitution
// amendment governance proposals calculated based on the participation EMA
func (k Keeper) GetConstitutionAmendmentQuorum(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	minQuorum := math.LegacyMustNewDecFromStr(params.MinConstitutionAmendmentQuorum)
	maxQuorum := math.LegacyMustNewDecFromStr(params.MaxConstitutionAmendmentQuorum)
	return k.getQuorum(ctx, minQuorum, maxQuorum)
}

// GetLawQuorum returns the dynamic quorum for law governance proposals
// calculated based on the participation EMA
func (k Keeper) GetLawQuorum(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	minQuorum := math.LegacyMustNewDecFromStr(params.MinLawQuorum)
	maxQuorum := math.LegacyMustNewDecFromStr(params.MaxLawQuorum)
	return k.getQuorum(ctx, minQuorum, maxQuorum)
}

// GetQuorum returns the dynamic quorum for governance proposals calculated based on the participation EMA
func (k Keeper) getQuorum(ctx sdk.Context, minQuorum, maxQuorum sdk.Dec) math.LegacyDec {
	participationEma := k.GetParticipationEMA(ctx)
	// quorum = min_quorum + (max_quorum - min_quorum) * participationEma
	return minQuorum.Add(maxQuorum.Sub(minQuorum).Mul(participationEma))
}
