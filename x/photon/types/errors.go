package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/photon module sentinel errors
var (
	ErrMintDisabled      = sdkerrors.Register(ModuleName, 1, "photon mint disabled")                             //nolint:staticcheck
	ErrBurnInvalidDenom  = sdkerrors.Register(ModuleName, 2, "invalid burned amount denom: expected bond denom") //nolint:staticcheck
	ErrNoMintablePhotons = sdkerrors.Register(ModuleName, 3, "no more photon can be minted")                     //nolint:staticcheck
	ErrNotEnoughPhotons  = sdkerrors.Register(ModuleName, 4, "not enough photon can be minted")                  //nolint:staticcheck

)