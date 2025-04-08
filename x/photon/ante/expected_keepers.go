package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

// PhotonKeeper defines the expected photon keeper.
type PhotonKeeper interface {
	GetParams(ctx sdk.Context) photontypes.Params
}
