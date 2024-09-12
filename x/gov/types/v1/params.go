package v1

import (
	"fmt"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Default period for deposits & voting
const (
	DefaultPeriod time.Duration = time.Hour * 24 * 2 // 2 days
)

// Default governance params
var (
	DefaultMinDepositTokens               = sdk.NewInt(10000000)
	DefaultQuorum                         = sdk.NewDecWithPrec(334, 3)
	DefaultThreshold                      = sdk.NewDecWithPrec(5, 1)
	DefaultVetoThreshold                  = sdk.NewDecWithPrec(334, 3)
	DefaultConstitutionAmendmentQuorum    = sdk.NewDecWithPrec(4, 1)
	DefaultConstitutionAmendmentThreshold = sdk.NewDecWithPrec(9, 1)
	DefaultLawQuorum                      = sdk.NewDecWithPrec(4, 1)
	DefaultLawThreshold                   = sdk.NewDecWithPrec(9, 1)
	DefaultMinInitialDepositRatio         = sdk.ZeroDec()
	DefaultBurnProposalPrevote            = false // set to false to replicate behavior of when this change was made (0.47)
	DefaultBurnVoteQuorom                 = false // set to false to  replicate behavior of when this change was made (0.47)
	DefaultBurnVoteVeto                   = true  // set to true to replicate behavior of when this change was made (0.47)
)

// Deprecated: NewDepositParams creates a new DepositParams object
func NewDepositParams(minDeposit sdk.Coins, maxDepositPeriod *time.Duration) DepositParams {
	return DepositParams{
		MinDeposit:       minDeposit,
		MaxDepositPeriod: maxDepositPeriod,
	}
}

// Deprecated: NewTallyParams creates a new TallyParams object
func NewTallyParams(quorum, threshold, vetoThreshold string) TallyParams {
	return TallyParams{
		Quorum:        quorum,
		Threshold:     threshold,
		VetoThreshold: vetoThreshold,
	}
}

// Deprecated: NewVotingParams creates a new VotingParams object
func NewVotingParams(votingPeriod *time.Duration) VotingParams {
	return VotingParams{
		VotingPeriod: votingPeriod,
	}
}

// NewParams creates a new Params instance with given values.
func NewParams(
	minDeposit sdk.Coins, maxDepositPeriod, votingPeriod time.Duration,
	quorum, threshold, vetoThreshold, constitutionAmendmentQuorum, constitutionAmendmentThreshold, lawQuorum, lawThreshold, minInitialDepositRatio string,
	burnProposalDeposit, burnVoteQuorum, burnVoteVeto bool,
) Params {
	return Params{
		MinDeposit:                     minDeposit,
		MaxDepositPeriod:               &maxDepositPeriod,
		VotingPeriod:                   &votingPeriod,
		Quorum:                         quorum,
		Threshold:                      threshold,
		VetoThreshold:                  vetoThreshold,
		ConstitutionAmendmentQuorum:    constitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold: constitutionAmendmentThreshold,
		LawQuorum:                      lawQuorum,
		LawThreshold:                   lawThreshold,
		MinInitialDepositRatio:         minInitialDepositRatio,
		BurnProposalDepositPrevote:     burnProposalDeposit,
		BurnVoteQuorum:                 burnVoteQuorum,
		BurnVoteVeto:                   burnVoteVeto,
	}
}

// DefaultParams returns the default governance params
func DefaultParams() Params {
	return NewParams(
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinDepositTokens)),
		DefaultPeriod,
		DefaultPeriod,
		DefaultQuorum.String(),
		DefaultThreshold.String(),
		DefaultVetoThreshold.String(),
		DefaultConstitutionAmendmentQuorum.String(),
		DefaultConstitutionAmendmentThreshold.String(),
		DefaultLawQuorum.String(),
		DefaultLawThreshold.String(),
		DefaultMinInitialDepositRatio.String(),
		DefaultBurnProposalPrevote,
		DefaultBurnVoteQuorom,
		DefaultBurnVoteVeto,
	)
}

// ValidateBasic performs basic validation on governance parameters.
func (p Params) ValidateBasic() error {
	if minDeposit := sdk.Coins(p.MinDeposit); minDeposit.Empty() || !minDeposit.IsValid() {
		return fmt.Errorf("invalid minimum deposit: %s", minDeposit)
	}

	if p.MaxDepositPeriod == nil {
		return fmt.Errorf("maximum deposit period must not be nil: %d", p.MaxDepositPeriod)
	}

	if p.MaxDepositPeriod.Seconds() <= 0 {
		return fmt.Errorf("maximum deposit period must be positive: %d", p.MaxDepositPeriod)
	}

	quorum, err := sdk.NewDecFromStr(p.Quorum)
	if err != nil {
		return fmt.Errorf("invalid quorum string: %w", err)
	}
	if quorum.IsNegative() {
		return fmt.Errorf("quorum must be positive: %s", quorum)
	}
	if quorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorum too large: %s", quorum)
	}

	threshold, err := sdk.NewDecFromStr(p.Threshold)
	if err != nil {
		return fmt.Errorf("invalid threshold string: %w", err)
	}
	if !threshold.IsPositive() {
		return fmt.Errorf("vote threshold must be positive: %s", threshold)
	}
	if threshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("vote threshold too large: %s", threshold)
	}

	vetoThreshold, err := sdk.NewDecFromStr(p.VetoThreshold)
	if err != nil {
		return fmt.Errorf("invalid vetoThreshold string: %w", err)
	}
	if !vetoThreshold.IsPositive() {
		return fmt.Errorf("veto threshold must be positive: %s", vetoThreshold)
	}
	if vetoThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("veto threshold too large: %s", vetoThreshold)
	}

	amendmentQuorum, err := sdk.NewDecFromStr(p.ConstitutionAmendmentQuorum)
	if err != nil {
		return fmt.Errorf("invalid constitution amendment quorum string: %w", err)
	}
	if amendmentQuorum.IsNegative() {
		return fmt.Errorf("constitution amendment quorum must be positive: %s", amendmentQuorum)
	}
	if amendmentQuorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("constitution amendment quorum too large: %s", amendmentQuorum)
	}
	if amendmentQuorum.LT(quorum) {
		return fmt.Errorf("constitution amendment quorum must be greater than or equal to governance quorum: %s", amendmentQuorum)
	}

	amendmentThreshold, err := sdk.NewDecFromStr(p.ConstitutionAmendmentThreshold)
	if err != nil {
		return fmt.Errorf("invalid constitution amendment threshold string: %w", err)
	}
	if !amendmentThreshold.IsPositive() {
		return fmt.Errorf("constitution amendment threshold must be positive: %s", amendmentThreshold)
	}
	if amendmentThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("constitution amendment threshold too large: %s", amendmentThreshold)
	}
	if amendmentThreshold.LT(threshold) {
		return fmt.Errorf("constitution amendment threshold must be greater than or equal to governance threshold: %s", amendmentThreshold)
	}

	lawQuorum, err := sdk.NewDecFromStr(p.LawQuorum)
	if err != nil {
		return fmt.Errorf("invalid law quorum string: %w", err)
	}
	if lawQuorum.IsNegative() {
		return fmt.Errorf("law quorum must be positive: %s", lawQuorum)
	}
	if lawQuorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("law quorum too large: %s", lawQuorum)
	}
	if lawQuorum.LT(quorum) {
		return fmt.Errorf("law quorum must be greater than or equal to governance quorum: %s", lawQuorum)
	}
	if lawQuorum.GT(amendmentQuorum) {
		return fmt.Errorf("law quorum must be less than or equal to constitution amendment quorum: %s", lawQuorum)
	}

	lawThreshold, err := sdk.NewDecFromStr(p.LawThreshold)
	if err != nil {
		return fmt.Errorf("invalid law threshold string: %w", err)
	}
	if !lawThreshold.IsPositive() {
		return fmt.Errorf("law threshold must be positive: %s", lawThreshold)
	}
	if lawThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("law threshold too large: %s", lawThreshold)
	}
	if lawThreshold.LT(threshold) {
		return fmt.Errorf("law threshold must be greater than or equal to governance threshold: %s", lawThreshold)
	}
	if lawThreshold.GT(amendmentThreshold) {
		return fmt.Errorf("law threshold must be less than or equal to constitution amendment threshold: %s", lawThreshold)
	}

	if p.VotingPeriod == nil {
		return fmt.Errorf("voting period must not be nil: %d", p.VotingPeriod)
	}

	if p.VotingPeriod.Seconds() <= 0 {
		return fmt.Errorf("voting period must be positive: %s", p.VotingPeriod)
	}

	minInitialDepositRatio, err := math.LegacyNewDecFromStr(p.MinInitialDepositRatio)
	if err != nil {
		return fmt.Errorf("invalid mininum initial deposit ratio of proposal: %w", err)
	}
	if minInitialDepositRatio.IsNegative() {
		return fmt.Errorf("mininum initial deposit ratio of proposal must be positive: %s", minInitialDepositRatio)
	}
	if minInitialDepositRatio.GT(math.LegacyOneDec()) {
		return fmt.Errorf("mininum initial deposit ratio of proposal is too large: %s", minInitialDepositRatio)
	}

	return nil
}
