package errors

import (
	"github.com/cosmos/cosmos-sdk/types/errors"
)

// groupCodespace is the codespace for all errors defined in group package
const groupCodespace = "group"

var (
	ErrEmpty        = errors.Register(groupCodespace, 2, "value is empty")  //nolint:staticcheck // SA1019
	ErrDuplicate    = errors.Register(groupCodespace, 3, "duplicate value") //nolint:staticcheck // SA1019
	ErrMaxLimit     = errors.Register(groupCodespace, 4, "limit exceeded")  //nolint:staticcheck // SA1019
	ErrType         = errors.Register(groupCodespace, 5, "invalid type")    //nolint:staticcheck // SA1019
	ErrInvalid      = errors.Register(groupCodespace, 6, "invalid value")   //nolint:staticcheck // SA1019
	ErrUnauthorized = errors.Register(groupCodespace, 7, "unauthorized")    //nolint:staticcheck // SA1019
	ErrModified     = errors.Register(groupCodespace, 8, "modified")        //nolint:staticcheck // SA1019
	ErrExpired      = errors.Register(groupCodespace, 9, "expired")         //nolint:staticcheck // SA1019
)
