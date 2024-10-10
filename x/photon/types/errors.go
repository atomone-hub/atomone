package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/photon module sentinel errors
var (
	ErrMintDisabled      = sdkerrors.Register(ModuleName, 1, "photon mint disabled")
	ErrBurnInvalidDenom  = sdkerrors.Register(ModuleName, 2, "invalid burned amount denom: expected bond denom")
	ErrNoMintablePhotons = sdkerrors.Register(ModuleName, 3, "no more photon can be minted")
)
