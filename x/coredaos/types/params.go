package types

import (
	fmt "fmt"
	time "time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewParams creates a new Params instance
func NewParams(steeringDaoAddress, oversightDaoAddress string, votingPeriodExtensionsLimit uint32, votingPeriodExtensionDuration time.Duration) Params {
	return Params{
		SteeringDaoAddress:            steeringDaoAddress,
		OversightDaoAddress:           oversightDaoAddress,
		VotingPeriodExtensionsLimit:   votingPeriodExtensionsLimit,
		VotingPeriodExtensionDuration: &votingPeriodExtensionDuration,
	}
}

const (
	// DefaultSteeringDaoAddress is the default address for the Steering DAO
	// An empty string indicates that no default address is set (disabled)
	DefaultSteeringDaoAddress = ""
	// DefaultOversightDaoAddress is the default address for the Oversight DAO
	// An empty string indicates that no default address is set (disabled)
	DefaultOversightDaoAddress = ""
	// DefaultVotingPeriodExtensionsLimit is the default limit for voting period extensions
	DefaultVotingPeriodExtensionsLimit = 3
	// DefaultVotingPeriodExtensionDuration is the default duration for voting period extensions
	DefaultVotingPeriodExtensionDuration = time.Hour * 24 * 7 // 7 days
)

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultSteeringDaoAddress,
		DefaultOversightDaoAddress,
		DefaultVotingPeriodExtensionsLimit,
		DefaultVotingPeriodExtensionDuration,
	)
}

// Validate validates the set of params
func (p Params) ValidateBasic() error {
	// Steering DAO address can only be empty (disabled) or a valid Bech32 address
	if p.SteeringDaoAddress != "" {
		if _, err := sdk.AccAddressFromBech32(p.SteeringDaoAddress); err != nil {
			return fmt.Errorf("invalid steering DAO address: %s: %w", p.SteeringDaoAddress, err)
		}
	}

	// Oversight DAO address can only be empty (disabled) or a valid Bech32 address
	if p.OversightDaoAddress != "" {
		if _, err := sdk.AccAddressFromBech32(p.OversightDaoAddress); err != nil {
			return fmt.Errorf("invalid oversight DAO address: %s: %w", p.OversightDaoAddress, err)
		}
	}

	// VotingPeriodExtensionDuration must be a positive duration
	if p.VotingPeriodExtensionDuration == nil {
		return fmt.Errorf("voting period extension duration must not be nil")
	}
	if p.VotingPeriodExtensionDuration.Seconds() <= 0 {
		return fmt.Errorf("voting period extension duration must be positive: %s", p.VotingPeriodExtensionDuration)
	}
	return nil
}
