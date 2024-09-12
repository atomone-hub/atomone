// DONTCOVER
package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/evidence module sentinel errors
var (
	ErrNoEvidenceHandlerExists = sdkerrors.Register(ModuleName, 2, "unregistered handler for evidence type") //nolint:staticcheck // SA1019
	ErrInvalidEvidence         = sdkerrors.Register(ModuleName, 3, "invalid evidence")                       //nolint:staticcheck // SA1019
	ErrNoEvidenceExists        = sdkerrors.Register(ModuleName, 4, "evidence does not exist")                //nolint:staticcheck // SA1019
	ErrEvidenceExists          = sdkerrors.Register(ModuleName, 5, "evidence already exists")                //nolint:staticcheck // SA1019
)
