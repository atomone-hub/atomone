package types

import (
	errorsmod "cosmossdk.io/errors"
)

// x/photon module sentinel errors
var (
	ErrAnnotationAlreadyPresent = errorsmod.Register(ModuleName, 1, "annotation already present")
	ErrProposalAlreadyEndorsed  = errorsmod.Register(ModuleName, 2, "proposal already endorsed")
	ErrFunctionDisabled         = errorsmod.Register(ModuleName, 3, "function is disabled")
	ErrCannotStake              = errorsmod.Register(ModuleName, 4, "core DAOs cannot stake")
)
