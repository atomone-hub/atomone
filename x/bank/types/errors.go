package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/bank module sentinel errors
var (
	ErrNoInputs              = sdkerrors.Register(ModuleName, 2, "no inputs to send transaction")   //nolint:staticcheck // SA1019
	ErrNoOutputs             = sdkerrors.Register(ModuleName, 3, "no outputs to send transaction")  //nolint:staticcheck // SA1019
	ErrInputOutputMismatch   = sdkerrors.Register(ModuleName, 4, "sum inputs != sum outputs")       //nolint:staticcheck // SA1019
	ErrSendDisabled          = sdkerrors.Register(ModuleName, 5, "send transactions are disabled")  //nolint:staticcheck // SA1019
	ErrDenomMetadataNotFound = sdkerrors.Register(ModuleName, 6, "client denom metadata not found") //nolint:staticcheck // SA1019
	ErrInvalidKey            = sdkerrors.Register(ModuleName, 7, "invalid key")                     //nolint:staticcheck // SA1019
	ErrDuplicateEntry        = sdkerrors.Register(ModuleName, 8, "duplicate entry")                 //nolint:staticcheck // SA1019
	ErrMultipleSenders       = sdkerrors.Register(ModuleName, 9, "multiple senders not allowed")    //nolint:staticcheck // SA1019
)
