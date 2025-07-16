package types

import (
	context "context"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

type ConsensusParamsKeeper interface {
	Get(context.Context) (tmproto.ConsensusParams, error)
}
