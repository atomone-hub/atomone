package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/photon module sentinel errors
var (
	ErrMintDisabled = sdkerrors.Register(ModuleName, 1, "photo mint disabled")
)
