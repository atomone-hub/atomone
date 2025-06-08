package types

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/gov module sentinel errors
var (
	ErrInvalidConstitutionAmendment = sdkerrors.Register(ModuleName, 170, "invalid constitution amendment")
	ErrUnknownProposal              = sdkerrors.Register(ModuleName, 180, "unknown proposal")
)
