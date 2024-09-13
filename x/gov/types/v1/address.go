package v1

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Address = GovernorAddress{}
)

type GovernorAddress []byte
