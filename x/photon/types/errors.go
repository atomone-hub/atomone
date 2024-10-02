package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/photon module sentinel errors
var (
	ErrMintDisabled     = sdkerrors.Register(ModuleName, 1, "photon mint disabled")
	ErrMintInvalidDenom = sdkerrors.Register(ModuleName, 2, "invalid burned amount denom: expected bond denom")
)
