package spectre

import (
	"crypto/sha256"
	"encoding/binary"
)

// This file implements the SSZ hash-tree-root primitives used by the
// Ethereum consensus specs, matching the semantics of the tree_hash crate
// vendored in spectre/packages/ethereum/tree_hash. Only the subset needed by
// the light client is implemented: containers of field roots, byte vectors,
// byte lists, uint64/uint256 chunks, and pubkey vectors.

// sha256Hash pairs two 32-byte chunks.
func sha256Hash(a, b [32]byte) [32]byte {
	h := sha256.New()
	h.Write(a[:])
	h.Write(b[:])
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// nextPow2 returns the next power of two of v (v itself if a power of two,
// 1 if v == 0).
func nextPow2(v uint64) uint64 {
	if v == 0 {
		return 1
	}
	p := uint64(1)
	for p < v {
		p <<= 1
	}
	return p
}

// merkleize computes the root of a merkle tree over the given leaves, padded
// with zero chunks up to next_pow2(len(leaves)) leaves. It matches
// tree_hash::MerkleHasher::with_leaves(len).finish(): an empty input yields
// the zero chunk, a single leaf is its own root.
func merkleize(leaves [][32]byte) [32]byte {
	switch len(leaves) {
	case 0:
		return [32]byte{}
	case 1:
		return leaves[0]
	}

	size := int(nextPow2(uint64(len(leaves))))
	level := make([][32]byte, size)
	copy(level, leaves)
	// zero-pad with [32]byte{} which is the Go zero value

	for len(level) > 1 {
		next := make([][32]byte, len(level)/2)
		for i := range next {
			next[i] = sha256Hash(level[2*i], level[2*i+1])
		}
		level = next
	}
	return level[0]
}

// mixInLength mixes the length (in bytes) into the merkle root, as specified
// by the SSZ spec for lists.
func mixInLength(root [32]byte, length int) [32]byte {
	var lenChunk [32]byte
	binary.LittleEndian.PutUint64(lenChunk[:8], uint64(length))
	return sha256Hash(root, lenChunk)
}

// uint64Chunk right-pads the little-endian encoding of v to 32 bytes.
func uint64Chunk(v uint64) [32]byte {
	var chunk [32]byte
	binary.LittleEndian.PutUint64(chunk[:8], v)
	return chunk
}

// uint256ChunkLE returns v (32 bytes, any byte order accepted) as a chunk in
// little-endian order. uint256 values hash as their little-endian bytes.
func uint256ChunkLE(be [32]byte) [32]byte {
	var le [32]byte
	for i := 0; i < 32; i++ {
		le[i] = be[31-i]
	}
	return le
}

// rightPad returns b right-padded with zeros to 32 bytes.
func rightPad(b []byte) [32]byte {
	var chunk [32]byte
	copy(chunk[:], b)
	return chunk
}

// packBytes splits b into 32-byte chunks, right-padding the last chunk.
func packBytes(b []byte) [][32]byte {
	if len(b) == 0 {
		return nil
	}
	chunks := make([][32]byte, (len(b)+31)/32)
	for i := range chunks {
		end := (i + 1) * 32
		if end > len(b) {
			end = len(b)
		}
		copy(chunks[i][:], b[i*32:end])
	}
	return chunks
}

// bytesVectorRoot hashes a fixed-size byte vector of up to 32 bytes: the
// value right-padded to a chunk.
func bytesVectorRoot(b []byte) [32]byte {
	return rightPad(b)
}

// byteListRoot hashes a variable-length byte list: merkleize(pack(bytes))
// with the length mixed in.
func byteListRoot(b []byte) [32]byte {
	return mixInLength(merkleize(packBytes(b)), len(b))
}

// fixedBytesVectorRoot hashes a fixed-size byte vector longer than 32 bytes
// (e.g. a 256-byte logs bloom): merkleize over the packed chunks without a
// length mix-in.
func fixedBytesVectorRoot(b []byte) [32]byte {
	return merkleize(packBytes(b))
}

// blsPubkeyRoot returns the hash tree root of a 48-byte BLS public key:
// the two packed chunks (32 + 16 padded bytes) hashed together. Keys of
// the wrong length hash as their zero-padded encoding, which fails
// verification against well-formed counterparties.
func blsPubkeyRoot(pk []byte) [32]byte {
	var padded [48]byte
	copy(padded[:], pk)
	return sha256Hash(rightPad(padded[:32]), rightPad(padded[32:48]))
}

// pubkeyVectorRoot returns the hash tree root of a vector of 48-byte BLS
// public keys: a merkle tree over the per-key roots, padded to the next
// power of two, without a length mix-in (vector semantics).
func pubkeyVectorRoot(pubkeys [][]byte) [32]byte {
	leaves := make([][32]byte, len(pubkeys))
	for i, pk := range pubkeys {
		leaves[i] = blsPubkeyRoot(pk)
	}
	return merkleize(leaves)
}

// containerRoot hashes a container from its field roots.
func containerRoot(fields [][32]byte) [32]byte {
	return merkleize(fields)
}

// beaconBlockHeaderRoot returns the hash tree root of a beacon block header.
func beaconBlockHeaderRoot(h *BeaconBlockHeader) [32]byte {
	return containerRoot([][32]byte{
		uint64Chunk(h.Slot),
		uint64Chunk(h.ProposerIndex),
		rightPad(h.ParentRoot),
		rightPad(h.StateRoot),
		rightPad(h.BodyRoot),
	})
}

// executionPayloadHeaderRoot returns the hash tree root of an execution
// payload header, per the Electra consensus specs.
func executionPayloadHeaderRoot(e *ExecutionPayloadHeader) [32]byte {
	return containerRoot([][32]byte{
		rightPad(e.ParentHash),                   // 1  parent_hash
		bytesVectorRoot(e.FeeRecipient),          // 2  fee_recipient (bytes20)
		rightPad(e.StateRoot),                    // 3  state_root
		rightPad(e.ReceiptsRoot),                 // 4  receipts_root
		fixedBytesVectorRoot(e.LogsBloom),        // 5 logs_bloom (bytes256)
		rightPad(e.PrevRandao),                   // 6  prev_randao
		uint64Chunk(e.BlockNumber),               // 7  block_number
		uint64Chunk(e.GasLimit),                  // 8  gas_limit
		uint64Chunk(e.GasUsed),                   // 9  gas_used
		uint64Chunk(e.Timestamp),                 // 10 timestamp
		byteListRoot(e.ExtraData),                // 11 extra_data (byte list)
		uint256ChunkLE(bytes32(e.BaseFeePerGas)), // 12 base_fee_per_gas (uint256 LE)
		rightPad(e.BlockHash),                    // 13 block_hash
		rightPad(e.TransactionsRoot),             // 14 transactions_root
		rightPad(e.WithdrawalsRoot),              // 15 withdrawals_root
		uint64Chunk(e.BlobGasUsed),               // 16 blob_gas_used
		uint64Chunk(e.ExcessBlobGas),             // 17 excess_blob_gas
	})
}

// syncCommitteeRoot returns the hash tree root of a sync committee: a
// container of the pubkeys vector root and the aggregate pubkey root.
func syncCommitteeRoot(sc *SyncCommittee) [32]byte {
	return containerRoot([][32]byte{
		pubkeyVectorRoot(sc.Pubkeys),
		blsPubkeyRoot(sc.AggregatePubkey),
	})
}

// signingRoot computes the consensus-spec signing root of an object root
// under a signature domain.
func signingRoot(objectRoot, domain [32]byte) [32]byte {
	return containerRoot([][32]byte{objectRoot, domain})
}

// forkDataRoot computes the fork data root for a fork version and the
// genesis validators root, used to derive signature domains.
func forkDataRoot(version []byte, genesisValidatorsRoot []byte) [32]byte {
	return containerRoot([][32]byte{
		rightPad(version[:min(4, len(version))]),
		rightPad(genesisValidatorsRoot[:min(32, len(genesisValidatorsRoot))]),
	})
}
