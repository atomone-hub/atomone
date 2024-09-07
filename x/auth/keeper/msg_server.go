package keeper

import (
	"context"

	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/auth/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	AccountKeeper
}

// NewMsgServerImpl returns an implementation of the x/auth MsgServer interface.
func NewMsgServerImpl(ak AccountKeeper) types.MsgServer {
	return &msgServer{
		AccountKeeper: ak,
	}
}

func (ms msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := ms.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
