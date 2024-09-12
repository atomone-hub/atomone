package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/crisis module sentinel errors
var (
	ErrNoSender         = sdkerrors.Register(ModuleName, 2, "sender address is empty") //nolint:staticcheck // SA1019
	ErrUnknownInvariant = sdkerrors.Register(ModuleName, 3, "unknown invariant")       //nolint:staticcheck // SA1019
)
