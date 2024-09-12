package proposal

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/params module sentinel errors
var (
	ErrUnknownSubspace  = sdkerrors.Register(ModuleName, 2, "unknown subspace")                      //nolint:staticcheck // SA1019
	ErrSettingParameter = sdkerrors.Register(ModuleName, 3, "failed to set parameter")               //nolint:staticcheck // SA1019
	ErrEmptyChanges     = sdkerrors.Register(ModuleName, 4, "submitted parameter changes are empty") //nolint:staticcheck // SA1019
	ErrEmptySubspace    = sdkerrors.Register(ModuleName, 5, "parameter subspace is empty")           //nolint:staticcheck // SA1019
	ErrEmptyKey         = sdkerrors.Register(ModuleName, 6, "parameter key is empty")                //nolint:staticcheck // SA1019
	ErrEmptyValue       = sdkerrors.Register(ModuleName, 7, "parameter value is empty")              //nolint:staticcheck // SA1019
)
