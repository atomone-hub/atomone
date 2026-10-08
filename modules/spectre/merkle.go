package spectre

import (
	"math/bits"

	errorsmod "cosmossdk.io/errors"
)

// Generalized indices of beacon state fields, per the Electra light client
// sync protocol specs.
const (
	// finalizedRootGindex is get_generalized_index(BeaconState,
	// 'finalized_checkpoint', 'root').
	finalizedRootGindex = 169
	// nextSyncCommitteeGindex is
	// get_generalized_index(BeaconState, 'next_sync_committee').
	nextSyncCommitteeGindex = 87
	// executionPayloadGindex is
	// get_generalized_index(BeaconBlockBody, 'execution_payload').
	executionPayloadGindex = 25
)

// floorLog2 returns floor(log2(n)) for n >= 1.
func floorLog2(n uint64) int {
	return bits.Len64(n) - 1
}

// subtreeIndex returns the index of a generalized index within its subtree,
// per the light client sync protocol: idx % 2^floor(log2(idx)).
func subtreeIndex(idx uint64) uint64 {
	return idx % (uint64(1) << uint(floorLog2(idx)))
}

// validateMerkleBranch verifies leaf against root with the given branch of
// depth floor(log2(gindex)) and subtree index, per the consensus specs
// (is_valid_merkle_branch). The walk consumes at most depth branch nodes.
func validateMerkleBranch(leaf [32]byte, branch [][32]byte, gindex uint64, root [32]byte) error {
	depth := floorLog2(gindex)
	if len(branch) < depth {
		return errorsmod.Wrapf(ErrInvalidMerkleBranch,
			"branch length %d is shorter than depth %d for gindex %d", len(branch), depth, gindex)
	}
	index := subtreeIndex(gindex)

	value := leaf
	for i := 0; i < depth; i++ {
		node := branch[i]
		if index>>uint(i)&1 == 1 {
			value = sha256Hash(node, value)
		} else {
			value = sha256Hash(value, node)
		}
	}

	if value != root {
		return errorsmod.Wrapf(ErrInvalidMerkleBranch,
			"computed root %X does not match expected root %X (leaf %X, depth %d, index %d)",
			value, root, leaf, depth, index)
	}
	return nil
}

// normalizeMerkleBranch left-pads a branch with zero chunks so its length
// equals the depth of the gindex. A branch longer than the depth is invalid.
// See the Electra light client fork specs.
func normalizeMerkleBranch(branch [][32]byte, gindex uint64) ([][32]byte, error) {
	depth := floorLog2(gindex)
	if len(branch) > depth {
		return nil, errorsmod.Wrapf(ErrInvalidMerkleBranch,
			"branch length %d exceeds depth %d for gindex %d", len(branch), depth, gindex)
	}
	numExtra := depth - len(branch)
	normalized := make([][32]byte, depth)
	copy(normalized[numExtra:], branch)
	return normalized, nil
}

// isValidNormalizedMerkleBranch validates a normalized merkle branch. Any
// padding entries must be zero.
func isValidNormalizedMerkleBranch(leaf [32]byte, normalizedBranch [][32]byte, gindex uint64, root [32]byte) error {
	depth := floorLog2(gindex)
	if len(normalizedBranch) < depth {
		return errorsmod.Wrapf(ErrInvalidMerkleBranch,
			"normalized branch length %d is shorter than depth %d", len(normalizedBranch), depth)
	}
	numExtra := len(normalizedBranch) - depth
	for i := 0; i < numExtra; i++ {
		if normalizedBranch[i] != ([32]byte{}) {
			return errorsmod.Wrap(ErrInvalidMerkleBranch, "normalized branch padding is non-zero")
		}
	}
	return validateMerkleBranch(leaf, normalizedBranch[numExtra:], gindex, root)
}
