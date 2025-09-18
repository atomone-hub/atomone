package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "coredaos"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

var ParamsKey = collections.NewPrefix(0)
