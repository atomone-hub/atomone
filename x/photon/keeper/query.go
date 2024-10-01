package keeper

import (
	"github.com/atomone-hub/atomone/x/photon/types"
)

var _ types.QueryServer = Keeper{}
