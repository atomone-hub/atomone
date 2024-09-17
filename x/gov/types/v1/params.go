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
	minVotingPeriod               = MinVotingPeriod
	DefaultMinDepositTokens       = sdk.NewInt(10000000)
	DefaultQuorum                 = sdk.NewDecWithPrec(25, 2)
	DefaultThreshold              = sdk.NewDecWithPrec(667, 3)
	DefaultMinInitialDepositRatio = sdk.ZeroDec()
	DefaultBurnProposalPrevote    = false                         // set to false to replicate behavior of when this change was made (0.47)
	DefaultBurnVoteQuorom         = false                         // set to false to  replicate behavior of when this change was made (0.47)
	DefaultMinDepositRatio        = sdk.MustNewDecFromStr("0.01") // NOTE: backport from v50
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
	quorum, threshold, minInitialDepositRatio string, burnProposalDeposit, burnVoteQuorum bool, minDepositRatio string,
) Params {
	return Params{
		MinDeposit:                 minDeposit,
		MaxDepositPeriod:           &maxDepositPeriod,
		VotingPeriod:               &votingPeriod,
		Quorum:                     quorum,
		Threshold:                  threshold,
		MinInitialDepositRatio:     minInitialDepositRatio,
		BurnProposalDepositPrevote: burnProposalDeposit,
		BurnVoteQuorum:             burnVoteQuorum,
		MinDepositRatio:            minDepositRatio,
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
		DefaultMinInitialDepositRatio.String(),
		DefaultBurnProposalPrevote,
		DefaultBurnVoteQuorom,
		DefaultMinDepositRatio.String(),
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
		return fmt.Errorf("quorom cannot be negative: %s", quorum)
	}
	if quorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorom too large: %s", p.Quorum)
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

	return nil
}
