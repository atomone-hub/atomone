package spectre

import (
	"math/bits"

	errorsmod "cosmossdk.io/errors"
)

// Generalized indices of beacon state and block body fields, per the light
// client sync protocol specs. They mirror
// packages/ethereum/types/src/consensus/merkle.rs in the spectre repository.
const (
	// finalizedRootGindexElectra is get_generalized_index(BeaconState,
	// 'finalized_checkpoint', 'root') from Electra through Fulu.
	finalizedRootGindexElectra = 169
	// nextSyncCommitteeGindexElectra is
	// get_generalized_index(BeaconState, 'next_sync_committee') from Electra
	// through Fulu.
	nextSyncCommitteeGindexElectra = 87
	// executionPayloadGindex is
	// get_generalized_index(BeaconBlockBody, 'execution_payload').
	executionPayloadGindex = 25

	// finalizedRootGindexGloas is the finalized root index from Gloas on
	// (consensus-specs gloas/light-client/sync-protocol.md#new-constants).
	finalizedRootGindexGloas = 735
	// nextSyncCommitteeGindexGloas is the next sync committee index from Gloas
	// on.
	nextSyncCommitteeGindexGloas = 2946
	// executionBlockHashGindexDeneb is get_generalized_index(
	// deneb.BeaconBlockBody, 'execution_payload', 'block_hash').
	executionBlockHashGindexDeneb = 812
	// executionBlockHashGindexGloas is get_generalized_index(BeaconBlockBody,
	// 'signed_execution_payload_bid', 'message', 'parent_block_hash').
	executionBlockHashGindexGloas = 2856
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
	for i := range depth {
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

// ensureBranchLength checks that a branch has exactly the depth of its
// generalized index. The light client spec types every branch with the exact
// length of its fork, so a branch of any other length is malformed rather than
// merely un-normalized.
func ensureBranchLength(name string, branch [][]byte, gindex uint64) error {
	if want := floorLog2(gindex); len(branch) != want {
		return errorsmod.Wrapf(ErrInvalidMerkleBranch, "%s has %d nodes, expected %d", name, len(branch), want)
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
	for i := range numExtra {
		if normalizedBranch[i] != ([32]byte{}) {
			return errorsmod.Wrap(ErrInvalidMerkleBranch, "normalized branch padding is non-zero")
		}
	}
	return validateMerkleBranch(leaf, normalizedBranch[numExtra:], gindex, root)
}
