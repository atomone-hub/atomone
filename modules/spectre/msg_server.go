package spectre

import (
	"context"
)

// msgServer implements the spectre module's Msg service.
type msgServer struct {
	keeper *Keeper
}

// NewMsgServer creates and returns a new spectre Msg server.
func NewMsgServer(keeper *Keeper) msgServer {
	return msgServer{keeper: keeper}
}

var _ MsgServer = msgServer{}

// MigrateForkParameters replaces the fork schedule of an active spectre
// client, through governance.
func (m msgServer) MigrateForkParameters(goCtx context.Context, msg *MsgMigrateForkParameters) (*MsgMigrateForkParametersResponse, error) {
	return m.keeper.MigrateForkParameters(goCtx, msg)
}
