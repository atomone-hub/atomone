package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/slashing module sentinel errors
var (
	ErrNoValidatorForAddress        = sdkerrors.Register(ModuleName, 2, "address is not associated with any known validator")                //nolint:staticcheck // SA1019
	ErrBadValidatorAddr             = sdkerrors.Register(ModuleName, 3, "validator does not exist for that address")                         //nolint:staticcheck // SA1019
	ErrValidatorJailed              = sdkerrors.Register(ModuleName, 4, "validator still jailed; cannot be unjailed")                        //nolint:staticcheck // SA1019
	ErrValidatorNotJailed           = sdkerrors.Register(ModuleName, 5, "validator not jailed; cannot be unjailed")                          //nolint:staticcheck // SA1019
	ErrMissingSelfDelegation        = sdkerrors.Register(ModuleName, 6, "validator has no self-delegation; cannot be unjailed")              //nolint:staticcheck // SA1019
	ErrSelfDelegationTooLowToUnjail = sdkerrors.Register(ModuleName, 7, "validator's self delegation less than minimum; cannot be unjailed") //nolint:staticcheck // SA1019
	ErrNoSigningInfoFound           = sdkerrors.Register(ModuleName, 8, "no validator signing info found")                                   //nolint:staticcheck // SA1019
)
