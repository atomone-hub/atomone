package types

import (
	"cosmossdk.io/errors"
)

// x/gov module sentinel errors
var (
	ErrUnknownProposal = errors.Register(ModuleName, 180, "unknown proposal")
)
