package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/capability module sentinel errors
var (
	ErrInvalidCapabilityName    = sdkerrors.Register(ModuleName, 2, "capability name not valid")              //nolint:staticcheck // SA1019
	ErrNilCapability            = sdkerrors.Register(ModuleName, 3, "provided capability is nil")             //nolint:staticcheck // SA1019
	ErrCapabilityTaken          = sdkerrors.Register(ModuleName, 4, "capability name already taken")          //nolint:staticcheck // SA1019
	ErrOwnerClaimed             = sdkerrors.Register(ModuleName, 5, "given owner already claimed capability") //nolint:staticcheck // SA1019
	ErrCapabilityNotOwned       = sdkerrors.Register(ModuleName, 6, "capability not owned by module")         //nolint:staticcheck // SA1019
	ErrCapabilityNotFound       = sdkerrors.Register(ModuleName, 7, "capability not found")                   //nolint:staticcheck // SA1019
	ErrCapabilityOwnersNotFound = sdkerrors.Register(ModuleName, 8, "owners not found for capability")        //nolint:staticcheck // SA1019
)
