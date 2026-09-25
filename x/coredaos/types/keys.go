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

var (
	// ParamsKey is the store prefix for the module parameters.
	ParamsKey = collections.NewPrefix(0)

	// VetoCleanupQueueKey is the store prefix for the veto cleanup queue, which
	// holds the proposals vetoed during the current block whose deposits and
	// votes still need to be cleaned up. The value is the BurnDeposit flag. The
	// queue is drained in the module EndBlocker within the same block, so it is
	// always empty at a block boundary.
	VetoCleanupQueueKey = collections.NewPrefix(1)
)
