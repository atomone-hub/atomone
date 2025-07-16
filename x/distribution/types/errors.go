package types

import "cosmossdk.io/errors"

// x/distribution module sentinel errors
var (
	ErrEmptyDelegatorAddr      = errors.Register(ModuleName, 102, "delegator address is empty")
	ErrEmptyWithdrawAddr       = errors.Register(ModuleName, 103, "withdraw address is empty")
	ErrEmptyValidatorAddr      = errors.Register(ModuleName, 104, "validator address is empty")
	ErrEmptyDelegationDistInfo = errors.Register(ModuleName, 105, "no delegation distribution info")
	ErrNoValidatorDistInfo     = errors.Register(ModuleName, 106, "no validator distribution info")
	ErrNoValidatorCommission   = errors.Register(ModuleName, 107, "no validator commission to withdraw")
	ErrSetWithdrawAddrDisabled = errors.Register(ModuleName, 108, "set withdraw address disabled")
	ErrBadDistribution         = errors.Register(ModuleName, 109, "community pool does not have sufficient coins to distribute")
	ErrInvalidProposalAmount   = errors.Register(ModuleName, 110, "invalid community pool spend proposal amount")
	ErrEmptyProposalRecipient  = errors.Register(ModuleName, 211, "invalid community pool spend proposal recipient")
	ErrNoValidatorExists       = errors.Register(ModuleName, 212, "validator does not exist")
	ErrNoDelegationExists      = errors.Register(ModuleName, 213, "delegation does not exist")
)
