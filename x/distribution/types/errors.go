package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/distribution module sentinel errors
var (
	ErrEmptyDelegatorAddr      = sdkerrors.Register(ModuleName, 2, "delegator address is empty")                                  //nolint:staticcheck // SA1019
	ErrEmptyWithdrawAddr       = sdkerrors.Register(ModuleName, 3, "withdraw address is empty")                                   //nolint:staticcheck // SA1019
	ErrEmptyValidatorAddr      = sdkerrors.Register(ModuleName, 4, "validator address is empty")                                  //nolint:staticcheck // SA1019
	ErrEmptyDelegationDistInfo = sdkerrors.Register(ModuleName, 5, "no delegation distribution info")                             //nolint:staticcheck // SA1019
	ErrNoValidatorDistInfo     = sdkerrors.Register(ModuleName, 6, "no validator distribution info")                              //nolint:staticcheck // SA1019
	ErrNoValidatorCommission   = sdkerrors.Register(ModuleName, 7, "no validator commission to withdraw")                         //nolint:staticcheck // SA1019
	ErrSetWithdrawAddrDisabled = sdkerrors.Register(ModuleName, 8, "set withdraw address disabled")                               //nolint:staticcheck // SA1019
	ErrBadDistribution         = sdkerrors.Register(ModuleName, 9, "community pool does not have sufficient coins to distribute") //nolint:staticcheck // SA1019
	ErrInvalidProposalAmount   = sdkerrors.Register(ModuleName, 10, "invalid community pool spend proposal amount")               //nolint:staticcheck // SA1019
	ErrEmptyProposalRecipient  = sdkerrors.Register(ModuleName, 11, "invalid community pool spend proposal recipient")            //nolint:staticcheck // SA1019
	ErrNoValidatorExists       = sdkerrors.Register(ModuleName, 12, "validator does not exist")                                   //nolint:staticcheck // SA1019
	ErrNoDelegationExists      = sdkerrors.Register(ModuleName, 13, "delegation does not exist")                                  //nolint:staticcheck // SA1019
)
