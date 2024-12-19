package v1

import (
	"fmt"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Default period for deposits & voting and min voting period
const (
	DefaultVotingPeriod  time.Duration = time.Hour * 24 * 21 // 21 days
	MinVotingPeriod      time.Duration = time.Hour * 24 * 21 // 21 days
	DefaultDepositPeriod time.Duration = time.Hour * 24 * 14 // 14 days
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

	DefaultQuorumTimeout                       time.Duration = DefaultVotingPeriod - (time.Hour * 24 * 1) // disabled by default (DefaultQuorumCheckCount must be set to a non-zero value to enable)
	DefaultMaxVotingPeriodExtension            time.Duration = DefaultVotingPeriod - DefaultQuorumTimeout // disabled by default (DefaultQuorumCheckCount must be set to a non-zero value to enable)
	DefaultQuorumCheckCount                    uint64        = 0                                          // disabled by default (0 means no check)
	DefaultMinDepositFloor                     sdk.Coins     = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinDepositTokens))
	DefaultMinDepositUpdatePeriod              time.Duration = time.Hour * 24 * 7
	DefaultMinDepositSensitivityTargetDistance uint64        = 2
	DefaultMinDepositIncreaseRatio                           = sdk.NewDecWithPrec(5, 2)
	DefaultMinDepositDecreaseRatio                           = sdk.NewDecWithPrec(25, 3)
	DefaultTargetActiveProposals               uint64        = 2
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
	// minDeposit sdk.Coins, // Deprecated in favor of dynamic min deposit
	maxDepositPeriod, votingPeriod time.Duration,
	quorum, threshold, constitutionAmendmentQuorum, constitutionAmendmentThreshold, lawQuorum, lawThreshold, minInitialDepositRatio string,
	burnProposalDeposit, burnVoteQuorum bool, minDepositRatio string,
	quorumTimeout, maxVotingPeriodExtension time.Duration, quorumCheckCount uint64,
	minDepositFloor sdk.Coins, minDepositUpdatePeriod time.Duration, minDepositSensitivityTargetDistance uint64,
	minDepositIncreaseRatio, minDepositDecreaseRatio string, targetActiveProposals uint64,
) Params {
	return Params{
		// MinDeposit:                     minDeposit, // Deprecated in favor of dynamic min deposit
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
		MinDepositThrottler: &MinDepositThrottler{
			FloorValue:                minDepositFloor,
			UpdatePeriod:              &minDepositUpdatePeriod,
			SensitivityTargetDistance: minDepositSensitivityTargetDistance,
			IncreaseRatio:             minDepositIncreaseRatio,
			DecreaseRatio:             minDepositDecreaseRatio,
			TargetActiveProposals:     targetActiveProposals,
		},
	}
}

// DefaultParams returns the default governance params
func DefaultParams() Params {
	return NewParams(
		// sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, DefaultMinDepositTokens)),
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
		DefaultMinDepositFloor,
		DefaultMinDepositUpdatePeriod,
		DefaultMinDepositSensitivityTargetDistance,
		DefaultMinDepositIncreaseRatio.String(),
		DefaultMinDepositDecreaseRatio.String(),
		DefaultTargetActiveProposals,
	)
}

// ValidateBasic performs basic validation on governance parameters.
func (p Params) ValidateBasic() error {
	// if minDeposit := sdk.Coins(p.MinDeposit); minDeposit.Empty() || !minDeposit.IsValid() {
	// 	return fmt.Errorf("invalid minimum deposit: %s", minDeposit)
	// }

	// if mindeposit is set, return error as it is deprecated
	// Q: is returning an error the best way to handle this? or perhaps just log a warning?
	//    after all this value is not used anymore in the codebase
	if p.MinDeposit != nil {
		return fmt.Errorf("manually setting min deposit is deprecated in favor of a dynamic min deposit")
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

	minDepositRatio, err := math.LegacyNewDecFromStr(p.MinDepositRatio)
	if err != nil {
		return fmt.Errorf("invalid mininum deposit ratio of proposal: %w", err)
	}
	if minDepositRatio.IsNegative() {
		return fmt.Errorf("mininum deposit ratio of proposal must be positive: %s", minDepositRatio)
	}
	if minDepositRatio.GT(math.LegacyOneDec()) {
		return fmt.Errorf("mininum deposit ratio of proposal is too large: %s", minDepositRatio)
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

	if minDepositFloor := sdk.Coins(p.MinDepositThrottler.FloorValue); minDepositFloor.Empty() || !minDepositFloor.IsValid() {
		return fmt.Errorf("invalid minimum deposit floor: %s", minDepositFloor)
	}

	if p.MinDepositThrottler.UpdatePeriod == nil {
		return fmt.Errorf("minimum deposit update period must not be nil: %d", p.MinDepositThrottler.UpdatePeriod)
	}

	if p.MinDepositThrottler.UpdatePeriod.Seconds() <= 0 {
		return fmt.Errorf("minimum deposit update period must be positive: %d", p.MinDepositThrottler.UpdatePeriod)
	}

	if p.MinDepositThrottler.UpdatePeriod.Seconds() > p.VotingPeriod.Seconds() {
		return fmt.Errorf("minimum deposit update period must be less than or equal to the voting period: %d", p.MinDepositThrottler.UpdatePeriod)
	}

	if p.MinDepositThrottler.SensitivityTargetDistance == 0 {
		return fmt.Errorf("minimum deposit sensitivity target distance must be positive: %d", p.MinDepositThrottler.SensitivityTargetDistance)
	}

	minDepositIncreaseRation, err := sdk.NewDecFromStr(p.MinDepositThrottler.IncreaseRatio)
	if err != nil {
		return fmt.Errorf("invalid minimum deposit increase ratio: %w", err)
	}
	if minDepositIncreaseRation.IsNegative() {
		return fmt.Errorf("minimum deposit increase ratio must be positive: %s", minDepositIncreaseRation)
	}
	if minDepositIncreaseRation.GT(math.LegacyOneDec()) {
		return fmt.Errorf("minimum deposit increase ratio too large: %s", minDepositIncreaseRation)
	}

	minDepositDecreaseRatio, err := sdk.NewDecFromStr(p.MinDepositThrottler.DecreaseRatio)
	if err != nil {
		return fmt.Errorf("invalid minimum deposit decrease ratio: %w", err)
	}
	if minDepositDecreaseRatio.IsNegative() {
		return fmt.Errorf("minimum deposit decrease ratio must be positive: %s", minDepositDecreaseRatio)
	}
	if minDepositDecreaseRatio.GT(math.LegacyOneDec()) {
		return fmt.Errorf("minimum deposit decrease ratio too large: %s", minDepositDecreaseRatio)
	}

	return nil
}
