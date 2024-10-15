package v1

import (
	"fmt"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Default period for deposits & voting and min voting period
const (
	DefaultVotingPeriod               time.Duration = time.Hour * 24 * 21 // 21 days
	MinVotingPeriod                   time.Duration = time.Hour * 24 * 21 // 21 days
	DefaultDepositPeriod              time.Duration = time.Hour * 24 * 14 // 14 days
	DefaultGovernorStatusChangePeriod time.Duration = time.Hour * 24 * 28 // 28 days
)

// Default governance params
var (
	minVotingPeriod                       = MinVotingPeriod
	DefaultMinDepositTokens               = sdk.NewInt(10000000)
	DefaultQuorum                         = sdk.NewDecWithPrec(25, 2)
	DefaultThreshold                      = sdk.NewDecWithPrec(667, 3)
	DefaultConstitutionAmendmentQuorum    = sdk.NewDecWithPrec(25, 2)
	DefaultConstitutionAmendmentThreshold = sdk.NewDecWithPrec(9, 1)
	DefaultLawQuorum                      = sdk.NewDecWithPrec(25, 2)
	DefaultLawThreshold                   = sdk.NewDecWithPrec(9, 1)
	DefaultMinInitialDepositRatio         = sdk.ZeroDec()
	DefaultBurnProposalPrevote            = false                    // set to false to replicate behavior of when this change was made (0.47)
	DefaultBurnVoteQuorom                 = false                    // set to false to  replicate behavior of when this change was made (0.47)
	DefaultMinDepositRatio                = sdk.NewDecWithPrec(1, 2) // NOTE: backport from v50

	DefaultQuorumTimeout             time.Duration = DefaultVotingPeriod - (time.Hour * 24 * 1) // disabled by default (DefaultQuorumCheckCount must be set to a non-zero value to enable)
	DefaultMaxVotingPeriodExtension  time.Duration = DefaultVotingPeriod - DefaultQuorumTimeout // disabled by default (DefaultQuorumCheckCount must be set to a non-zero value to enable)
	DefaultQuorumCheckCount          uint64        = 0                                          // disabled by default (0 means no check)
	DefaultMaxGovernors              uint64        = 100
	DefaultMinGovernorSelfDelegation               = sdk.NewInt(1000_000000)
)

// Deprecated: NewDepositParams creates a new DepositParams object
func NewDepositParams(minDeposit sdk.Coins, maxDepositPeriod *time.Duration) DepositParams {
	return DepositParams{
		MinDeposit:       minDeposit,
		MaxDepositPeriod: maxDepositPeriod,
	}
}

// Deprecated: NewTallyParams creates a new TallyParams object
func NewTallyParams(quorum, threshold string) TallyParams {
	return TallyParams{
		Quorum:    quorum,
		Threshold: threshold,
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
	quorum, threshold, constitutionAmendmentQuorum, constitutionAmendmentThreshold, lawQuorum, lawThreshold, minInitialDepositRatio string,
	burnProposalDeposit, burnVoteQuorum bool, minDepositRatio string,
	quorumTimeout, maxVotingPeriodExtension time.Duration, quorumCheckCount uint64,
	governorStatusChangePeriod time.Duration, minGovernorSelfDelegation string,
) Params {
	return Params{
		MinDeposit:                     minDeposit,
		MaxDepositPeriod:               &maxDepositPeriod,
		VotingPeriod:                   &votingPeriod,
		Quorum:                         quorum,
		Threshold:                      threshold,
		ConstitutionAmendmentQuorum:    constitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold: constitutionAmendmentThreshold,
		LawQuorum:                      lawQuorum,
		LawThreshold:                   lawThreshold,
		MinInitialDepositRatio:         minInitialDepositRatio,
		BurnProposalDepositPrevote:     burnProposalDeposit,
		BurnVoteQuorum:                 burnVoteQuorum,
		MinDepositRatio:                minDepositRatio,
		QuorumTimeout:                  &quorumTimeout,
		MaxVotingPeriodExtension:       &maxVotingPeriodExtension,
		QuorumCheckCount:               quorumCheckCount,
		GovernorStatusChangePeriod:     &governorStatusChangePeriod,
		MinGovernorSelfDelegation:      minGovernorSelfDelegation,
	}
}

// DefaultParams returns the default governance params
func DefaultParams() Params {
	return NewParams(
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinDepositTokens)),
		DefaultDepositPeriod,
		DefaultVotingPeriod,
		DefaultQuorum.String(),
		DefaultThreshold.String(),
		DefaultConstitutionAmendmentQuorum.String(),
		DefaultConstitutionAmendmentThreshold.String(),
		DefaultLawQuorum.String(),
		DefaultLawThreshold.String(),
		DefaultMinInitialDepositRatio.String(),
		DefaultBurnProposalPrevote,
		DefaultBurnVoteQuorom,
		DefaultMinDepositRatio.String(),
		DefaultQuorumTimeout,
		DefaultMaxVotingPeriodExtension,
		DefaultQuorumCheckCount,
		DefaultGovernorStatusChangePeriod,
		DefaultMinGovernorSelfDelegation.String(),
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
		return fmt.Errorf("voting period must not be nil")
	}

	if p.VotingPeriod.Seconds() < minVotingPeriod.Seconds() {
		return fmt.Errorf("voting period must be at least %s: %s", minVotingPeriod.String(), p.VotingPeriod.String())
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

	if p.QuorumCheckCount > 0 {
		// If quorum check is enabled, validate quorum check params
		if p.QuorumTimeout == nil {
			return fmt.Errorf("quorum timeout must not be nil: %d", p.QuorumTimeout)
		}
		if p.QuorumTimeout.Seconds() < 0 {
			return fmt.Errorf("quorum timeout must be 0 or greater: %s", p.QuorumTimeout)
		}
		if p.QuorumTimeout.Nanoseconds() >= p.VotingPeriod.Nanoseconds() {
			return fmt.Errorf("quorum timeout %s must be strictly less than the voting period %s", p.QuorumTimeout, p.VotingPeriod)
		}

		if p.MaxVotingPeriodExtension == nil {
			return fmt.Errorf("max voting period extension must not be nil: %d", p.MaxVotingPeriodExtension)
		}
		if p.MaxVotingPeriodExtension.Nanoseconds() < p.VotingPeriod.Nanoseconds()-p.QuorumTimeout.Nanoseconds() {
			return fmt.Errorf("max voting period extension %s must be greater than or equal to the difference between the voting period %s and the quorum timeout %s", p.MaxVotingPeriodExtension, p.VotingPeriod, p.QuorumTimeout)
		}
	}

	if p.GovernorStatusChangePeriod == nil {
		return fmt.Errorf("governor status change period must not be nil: %d", p.GovernorStatusChangePeriod)
	}

	if p.GovernorStatusChangePeriod.Seconds() <= 0 {
		return fmt.Errorf("governor status change period must be positive: %d", p.GovernorStatusChangePeriod)
	}

	minGovernorSelfDelegation, ok := math.NewIntFromString(p.MinGovernorSelfDelegation)
	if !ok {
		return fmt.Errorf("invalid minimum governor self delegation: %s", p.MinGovernorSelfDelegation)
	}
	if minGovernorSelfDelegation.IsNegative() {
		return fmt.Errorf("minimum governor self delegation must be positive: %s", minGovernorSelfDelegation)
	}

	return nil
}
