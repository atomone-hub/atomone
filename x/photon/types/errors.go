package types

import (
	errorsmod "cosmossdk.io/errors"
)

// x/photon module sentinel errors
var (
	ErrMintDisabled     = errorsmod.Register(ModuleName, 1, "photon mint disabled")
	ErrBurnInvalidDenom = errorsmod.Register(ModuleName, 2, "invalid burned amount denom: expected bond denom")
	ErrZeroMintPhotons  = errorsmod.Register(ModuleName, 3, "no mintable photon after rounding, try higher burn")
	ErrTooManyFeeCoins  = errorsmod.Register(ModuleName, 5, "too many fee coins, only accepts fees in one denom")
	ErrInvalidFeeToken  = errorsmod.Register(ModuleName, 6, "invalid fee token")
)
