package errors

import (
	errorsmod "github.com/atomone-hub/atomone/errors"
)

const codespace = "atomone"

var (
	Register = errorsmod.Register
)

var (
	// ErrTxDecode is returned if we cannot parse a transaction
	ErrTxDecode = Register(codespace, 1, "tx parse error")
	// ErrUnauthorized is used whenever a request without sufficient
	// authorization is handled.
	ErrUnauthorized = Register(codespace, 2, "unauthorized")

	// ErrInsufficientFunds is used when the account cannot pay requested amount.
	ErrInsufficientFunds = Register(codespace, 3, "insufficient funds")

	// ErrInsufficientFunds is used when the account cannot pay requested amount.
	ErrInsufficientFee = Register(codespace, 4, "insufficient fee")

	// ErrInvalidCoins is used when sdk.Coins are invalid.
	ErrInvalidCoins = Register(codespace, 5, "invalid coins")

	// ErrInvalidType defines an error an invalid type.
	ErrInvalidType = Register(codespace, 6, "invalid type")

	// ErrLogic defines an internal logic error, e.g. an invariant or assertion
	// that is violated. It is a programmer error, not a user-facing error.
	ErrLogic = Register(codespace, 7, "internal logic error")

	// ErrNotFound defines an error when requested entity doesn't exist in the state.
	ErrNotFound = Register(codespace, 8, "not found")

	// ErrInsufficientStake is used when the account has insufficient staked tokens.
	ErrInsufficientStake = Register(codespace, 9, "insufficient stake")
)
