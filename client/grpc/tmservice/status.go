package tmservice

import (
	"context"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/atomone-hub/atomone/client"
)

func getNodeStatus(ctx context.Context, clientCtx client.Context) (*coretypes.ResultStatus, error) {
	node, err := clientCtx.GetNode()
	if err != nil {
		return &coretypes.ResultStatus{}, err
	}
	return node.Status(ctx)
}
