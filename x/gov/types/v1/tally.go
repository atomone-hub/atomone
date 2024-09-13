package v1

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GovernorGovInfo used for tallying
type GovernorGovInfo struct {
	Address               GovernorAddress       // address of the governor
	Delegations           []ValidatorDelegation // Delegations of the governor
	DelegationsDeductions []ValidatorDelegation // Delegator deductions from validator's delegators voting independently
	Vote                  WeightedVoteOptions   // Vote of the validator
}

// NewGovernorGovInfo creates a GovernorGovInfo instance
func NewGovernorGovInfo(address GovernorAddress, delegations []ValidatorDelegation, deductions []ValidatorDelegation, options WeightedVoteOptions) GovernorGovInfo {
	return GovernorGovInfo{
		Address:               address,
		Delegations:           delegations,
		DelegationsDeductions: deductions,
		Vote:                  options,
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
