package types

import (
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ConsensusParamsKeeper interface {
	Get(sdk.Context) (*tmproto.ConsensusParams, error)
}
