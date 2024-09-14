package v1

import (
	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/x/gov/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GovernorGovInfo used for tallying
type GovernorGovInfo struct {
	Address             types.GovernorAddress // address of the governor
	ValShares           map[string]sdk.Dec    // shares held for each validator
	ValSharesDeductions map[string]sdk.Dec    // deductions from validator's shares when a delegator votes independently
	Vote                WeightedVoteOptions   // vote of the governor
}

// NewGovernorGovInfo creates a GovernorGovInfo instance
func NewGovernorGovInfo(address types.GovernorAddress, valShares []GovernorValShares, options WeightedVoteOptions) GovernorGovInfo {
	valSharesMap := make(map[string]sdk.Dec)
	for _, valShare := range valShares {
		valSharesMap[valShare.ValidatorAddress] = valShare.Shares
	}

	return GovernorGovInfo{
		Address:             address,
		ValShares:           valSharesMap,
		ValSharesDeductions: make(map[string]sdk.Dec),
		Vote:                options,
	}
}

// NewTallyResult creates a new TallyResult instance
func NewTallyResult(yes, abstain, no, noWithVeto math.Int) TallyResult {
	return TallyResult{
		YesCount:        yes.String(),
		AbstainCount:    abstain.String(),
		NoCount:         no.String(),
		NoWithVetoCount: noWithVeto.String(),
	}
}

// NewTallyResultFromMap creates a new TallyResult instance from a Option -> Dec map
func NewTallyResultFromMap(results map[VoteOption]sdk.Dec) TallyResult {
	return NewTallyResult(
		results[OptionYes].TruncateInt(),
		results[OptionAbstain].TruncateInt(),
		results[OptionNo].TruncateInt(),
		results[OptionNoWithVeto].TruncateInt(),
	)
}

// EmptyTallyResult returns an empty TallyResult.
func EmptyTallyResult() TallyResult {
	return NewTallyResult(math.ZeroInt(), math.ZeroInt(), math.ZeroInt(), math.ZeroInt())
}

// Equals returns if two tally results are equal.
func (tr TallyResult) Equals(comp TallyResult) bool {
	return tr.YesCount == comp.YesCount &&
		tr.AbstainCount == comp.AbstainCount &&
		tr.NoCount == comp.NoCount &&
		tr.NoWithVetoCount == comp.NoWithVetoCount
}
