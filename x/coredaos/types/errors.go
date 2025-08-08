package types

import (
	errorsmod "cosmossdk.io/errors"
)

// x/photon module sentinel errors
var (
	ErrInvalidSigner            = errorsmod.Register(ModuleName, 1, "expected core DAO account as only signer for this message")
	ErrAnnotationAlreadyPresent = errorsmod.Register(ModuleName, 2, "annotation already present")
	ErrProposalAlreadyEndorsed  = errorsmod.Register(ModuleName, 3, "proposal already endorsed")
	ErrFunctionDisabled         = errorsmod.Register(ModuleName, 4, "function is disabled")
	ErrCannotStake              = errorsmod.Register(ModuleName, 5, "core DAOs cannot stake")
)
