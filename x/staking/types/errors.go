package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/staking module sentinel errors
//
// TODO: Many of these errors are redundant. They should be removed and replaced
// by sdkerrors.ErrInvalidRequest.
//
// REF: https://github.com/cosmos/cosmos-sdk/issues/5450
var (
	ErrEmptyValidatorAddr              = sdkerrors.Register(ModuleName, 2, "empty validator address")                                                                                                          //nolint:staticcheck // SA1019
	ErrNoValidatorFound                = sdkerrors.Register(ModuleName, 3, "validator does not exist")                                                                                                         //nolint:staticcheck // SA1019
	ErrValidatorOwnerExists            = sdkerrors.Register(ModuleName, 4, "validator already exist for this operator address; must use new validator operator address")                                       //nolint:staticcheck // SA1019
	ErrValidatorPubKeyExists           = sdkerrors.Register(ModuleName, 5, "validator already exist for this pubkey; must use new validator pubkey")                                                           //nolint:staticcheck // SA1019
	ErrValidatorPubKeyTypeNotSupported = sdkerrors.Register(ModuleName, 6, "validator pubkey type is not supported")                                                                                           //nolint:staticcheck // SA1019
	ErrValidatorJailed                 = sdkerrors.Register(ModuleName, 7, "validator for this address is currently jailed")                                                                                   //nolint:staticcheck // SA1019
	ErrBadRemoveValidator              = sdkerrors.Register(ModuleName, 8, "failed to remove validator")                                                                                                       //nolint:staticcheck // SA1019
	ErrCommissionNegative              = sdkerrors.Register(ModuleName, 9, "commission must be positive")                                                                                                      //nolint:staticcheck // SA1019
	ErrCommissionHuge                  = sdkerrors.Register(ModuleName, 10, "commission cannot be more than 100%")                                                                                             //nolint:staticcheck // SA1019
	ErrCommissionGTMaxRate             = sdkerrors.Register(ModuleName, 11, "commission cannot be more than the max rate")                                                                                     //nolint:staticcheck // SA1019
	ErrCommissionUpdateTime            = sdkerrors.Register(ModuleName, 12, "commission cannot be changed more than once in 24h")                                                                              //nolint:staticcheck // SA1019
	ErrCommissionChangeRateNegative    = sdkerrors.Register(ModuleName, 13, "commission change rate must be positive")                                                                                         //nolint:staticcheck // SA1019
	ErrCommissionChangeRateGTMaxRate   = sdkerrors.Register(ModuleName, 14, "commission change rate cannot be more than the max rate")                                                                         //nolint:staticcheck // SA1019
	ErrCommissionGTMaxChangeRate       = sdkerrors.Register(ModuleName, 15, "commission cannot be changed more than max change rate")                                                                          //nolint:staticcheck // SA1019
	ErrSelfDelegationBelowMinimum      = sdkerrors.Register(ModuleName, 16, "validator's self delegation must be greater than their minimum self delegation")                                                  //nolint:staticcheck // SA1019
	ErrMinSelfDelegationDecreased      = sdkerrors.Register(ModuleName, 17, "minimum self delegation cannot be decrease")                                                                                      //nolint:staticcheck // SA1019
	ErrEmptyDelegatorAddr              = sdkerrors.Register(ModuleName, 18, "empty delegator address")                                                                                                         //nolint:staticcheck // SA1019
	ErrNoDelegation                    = sdkerrors.Register(ModuleName, 19, "no delegation for (address, validator) tuple")                                                                                    //nolint:staticcheck // SA1019
	ErrBadDelegatorAddr                = sdkerrors.Register(ModuleName, 20, "delegator does not exist with address")                                                                                           //nolint:staticcheck // SA1019
	ErrNoDelegatorForAddress           = sdkerrors.Register(ModuleName, 21, "delegator does not contain delegation")                                                                                           //nolint:staticcheck // SA1019
	ErrInsufficientShares              = sdkerrors.Register(ModuleName, 22, "insufficient delegation shares")                                                                                                  //nolint:staticcheck // SA1019
	ErrDelegationValidatorEmpty        = sdkerrors.Register(ModuleName, 23, "cannot delegate to an empty validator")                                                                                           //nolint:staticcheck // SA1019
	ErrNotEnoughDelegationShares       = sdkerrors.Register(ModuleName, 24, "not enough delegation shares")                                                                                                    //nolint:staticcheck // SA1019
	ErrNotMature                       = sdkerrors.Register(ModuleName, 25, "entry not mature")                                                                                                                //nolint:staticcheck // SA1019
	ErrNoUnbondingDelegation           = sdkerrors.Register(ModuleName, 26, "no unbonding delegation found")                                                                                                   //nolint:staticcheck // SA1019
	ErrMaxUnbondingDelegationEntries   = sdkerrors.Register(ModuleName, 27, "too many unbonding delegation entries for (delegator, validator) tuple")                                                          //nolint:staticcheck // SA1019
	ErrNoRedelegation                  = sdkerrors.Register(ModuleName, 28, "no redelegation found")                                                                                                           //nolint:staticcheck // SA1019
	ErrSelfRedelegation                = sdkerrors.Register(ModuleName, 29, "cannot redelegate to the same validator")                                                                                         //nolint:staticcheck // SA1019
	ErrTinyRedelegationAmount          = sdkerrors.Register(ModuleName, 30, "too few tokens to redelegate (truncates to zero tokens)")                                                                         //nolint:staticcheck // SA1019
	ErrBadRedelegationDst              = sdkerrors.Register(ModuleName, 31, "redelegation destination validator not found")                                                                                    //nolint:staticcheck // SA1019
	ErrTransitiveRedelegation          = sdkerrors.Register(ModuleName, 32, "redelegation to this validator already in progress; first redelegation to this validator must complete before next redelegation") //nolint:staticcheck // SA1019
	ErrMaxRedelegationEntries          = sdkerrors.Register(ModuleName, 33, "too many redelegation entries for (delegator, src-validator, dst-validator) tuple")                                               //nolint:staticcheck // SA1019
	ErrDelegatorShareExRateInvalid     = sdkerrors.Register(ModuleName, 34, "cannot delegate to validators with invalid (zero) ex-rate")                                                                       //nolint:staticcheck // SA1019
	ErrBothShareMsgsGiven              = sdkerrors.Register(ModuleName, 35, "both shares amount and shares percent provided")                                                                                  //nolint:staticcheck // SA1019
	ErrNeitherShareMsgsGiven           = sdkerrors.Register(ModuleName, 36, "neither shares amount nor shares percent provided")                                                                               //nolint:staticcheck // SA1019
	ErrInvalidHistoricalInfo           = sdkerrors.Register(ModuleName, 37, "invalid historical info")                                                                                                         //nolint:staticcheck // SA1019
	ErrNoHistoricalInfo                = sdkerrors.Register(ModuleName, 38, "no historical info found")                                                                                                        //nolint:staticcheck // SA1019
	ErrEmptyValidatorPubKey            = sdkerrors.Register(ModuleName, 39, "empty validator public key")                                                                                                      //nolint:staticcheck // SA1019
	ErrCommissionLTMinRate             = sdkerrors.Register(ModuleName, 40, "commission cannot be less than min rate")                                                                                         //nolint:staticcheck // SA1019
	ErrUnbondingNotFound               = sdkerrors.Register(ModuleName, 41, "unbonding operation not found")                                                                                                   //nolint:staticcheck // SA1019
	ErrUnbondingOnHoldRefCountNegative = sdkerrors.Register(ModuleName, 42, "cannot un-hold unbonding operation that is not on hold")                                                                          //nolint:staticcheck // SA1019
)
