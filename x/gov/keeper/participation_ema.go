package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// GetParticipationEMA returns the governance participation EMA
func (k Keeper) GetParticipationEMA(ctx sdk.Context) math.LegacyDec {
	return k.getParticipationEMAByKey(ctx, types.KeyParticipationEMA)
}

// GetConstitutionAmendmentParticipationEMA returns the governance
// participation EMA for constitution amendment proposals.
func (k Keeper) GetConstitutionAmendmentParticipationEMA(ctx sdk.Context) math.LegacyDec {
	return k.getParticipationEMAByKey(ctx, types.KeyConstitutionAmendmentParticipationEMA)
}

// GetLawParticipationEMA returns the governance participation EMA for law
// proposals.
func (k Keeper) GetLawParticipationEMA(ctx sdk.Context) math.LegacyDec {
	return k.getParticipationEMAByKey(ctx, types.KeyLawParticipationEMA)
}

func (k Keeper) getParticipationEMAByKey(ctx sdk.Context, key []byte) math.LegacyDec {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(key)
	if bz == nil {
		return math.LegacyDec{}
	}
	return math.LegacyMustNewDecFromStr(string(bz))
}

// SetParticipationEMA sets the governance participation EMA
func (k Keeper) SetParticipationEMA(ctx sdk.Context, participationEma math.LegacyDec) {
	k.setParticipationEMAByKey(ctx, types.KeyParticipationEMA, participationEma)
}

// SetConstitutionAmendmentParticipationEMA sets the governance participation
// EMA for a constitution amendment proposal.
func (k Keeper) SetConstitutionAmendmentParticipationEMA(ctx sdk.Context, participationEma math.LegacyDec) {
	k.setParticipationEMAByKey(ctx, types.KeyConstitutionAmendmentParticipationEMA, participationEma)
}

// SetLawParticipationEMA sets the governance participation EMA for a law
// proposal
func (k Keeper) SetLawParticipationEMA(ctx sdk.Context, participationEma math.LegacyDec) {
	k.setParticipationEMAByKey(ctx, types.KeyLawParticipationEMA, participationEma)
}

func (k Keeper) setParticipationEMAByKey(ctx sdk.Context, key []byte, participationEma math.LegacyDec) {
	store := ctx.KVStore(k.storeKey)
	store.Set(key, []byte(participationEma.String()))
}

// UpdateParticipationEMA updates the governance participation EMA
func (k Keeper) UpdateParticipationEMA(ctx sdk.Context, proposal v1.Proposal, participation math.LegacyDec) {
	kinds := k.ProposalKinds(proposal)
	if kinds.HasKindConstitutionAmendment() {
		k.updateParticipationEMAByKey(ctx, types.KeyConstitutionAmendmentParticipationEMA, participation)
	}
	if kinds.HasKindLaw() {
		k.updateParticipationEMAByKey(ctx, types.KeyLawParticipationEMA, participation)
	}
	if kinds.HasKindAny() {
		k.updateParticipationEMAByKey(ctx, types.KeyParticipationEMA, participation)
	}
}

func (k Keeper) updateParticipationEMAByKey(ctx sdk.Context, key []byte, participation math.LegacyDec) {
	old_participationEma := k.getParticipationEMAByKey(ctx, key)
	// new_participationEma = 0.8 * old_participationEma + 0.2 * participation
	new_participationEma := old_participationEma.Mul(sdk.NewDecWithPrec(8, 1)).Add(participation.Mul(sdk.NewDecWithPrec(2, 1)))
	k.setParticipationEMAByKey(ctx, key, new_participationEma)
}

// GetQuorum returns the dynamic quorum for governance proposals calculated
// based on the participation EMA
func (k Keeper) GetQuorum(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	participation := k.GetParticipationEMA(ctx)
	minQuorum := math.LegacyMustNewDecFromStr(params.MinQuorum)
	maxQuorum := math.LegacyMustNewDecFromStr(params.MaxQuorum)
	return computeQuorum(participation, minQuorum, maxQuorum)
}

// GetConstitutionAmendmentQuorum returns the dynamic quorum for constitution
// amendment governance proposals calculated based on the participation EMA
func (k Keeper) GetConstitutionAmendmentQuorum(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	participation := k.GetConstitutionAmendmentParticipationEMA(ctx)
	minQuorum := math.LegacyMustNewDecFromStr(params.MinConstitutionAmendmentQuorum)
	maxQuorum := math.LegacyMustNewDecFromStr(params.MaxConstitutionAmendmentQuorum)
	return computeQuorum(participation, minQuorum, maxQuorum)
}

// GetLawQuorum returns the dynamic quorum for law governance proposals
// calculated based on the participation EMA
func (k Keeper) GetLawQuorum(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	participation := k.GetLawParticipationEMA(ctx)
	minQuorum := math.LegacyMustNewDecFromStr(params.MinLawQuorum)
	maxQuorum := math.LegacyMustNewDecFromStr(params.MaxLawQuorum)
	return computeQuorum(participation, minQuorum, maxQuorum)
}

// computeQuorum returns the dynamic quorum for governance proposals calculated
// based on the participation EMA, min and max quorum.
func computeQuorum(participationEma, minQuorum, maxQuorum sdk.Dec) math.LegacyDec {
	// quorum = min_quorum + (max_quorum - min_quorum) * participationEma
	return minQuorum.Add(maxQuorum.Sub(minQuorum).Mul(participationEma))
}
