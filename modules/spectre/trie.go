package spectre

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	errorsmod "cosmossdk.io/errors"
)

// Proof size bounds, mirroring the spectre ethereum light client.
const (
	maxProofNodes     = 64
	maxProofNodeBytes = 32 * 1024
)

// ethAccount is the RLP encoding of an Ethereum account in the state trie.
type ethAccount struct {
	Nonce       uint64
	Balance     *big.Int
	StorageRoot common.Hash
	CodeHash    common.Hash
}

// validateProofBounds enforces DoS limits on submitted proof nodes before
// any hashing or decoding work.
func validateProofBounds(proofs ...[][]byte) error {
	for _, nodes := range proofs {
		if len(nodes) > maxProofNodes {
			return errorsmod.Wrapf(ErrInvalidProof, "too many proof nodes: %d > %d", len(nodes), maxProofNodes)
		}
		for i, node := range nodes {
			if len(node) > maxProofNodeBytes {
				return errorsmod.Wrapf(ErrInvalidProof, "proof node %d too large: %d > %d", i, len(node), maxProofNodeBytes)
			}
		}
	}
	return nil
}

// trieGet walks an Ethereum Merkle Patricia Trie from root following
// keccak256(key). It returns the value stored at key, or nil when the proof
// shows the key is absent. Every node in the walk must be present in the
// proof and must hash to its position in the trie.
func trieGet(root [32]byte, key []byte, proof [][]byte) ([]byte, error) {
	nodes := make(map[common.Hash][]byte, len(proof))
	for _, node := range proof {
		if len(node) == 0 {
			return nil, errorsmod.Wrap(ErrInvalidProof, "empty proof node")
		}
		nodes[crypto.Keccak256Hash(node)] = node
	}

	node, ok := nodes[common.BytesToHash(root[:])]
	if !ok {
		return nil, errorsmod.Wrap(ErrInvalidProof, "proof is missing the root node")
	}

	nibbles := keyToNibbles(crypto.Keccak256(key))
	idx := 0

	for {
		var decoded interface{}
		if err := rlp.DecodeBytes(node, &decoded); err != nil {
			return nil, errorsmod.Wrap(ErrInvalidProof, "invalid RLP node encoding")
		}
		list, ok := decoded.([]interface{})
		if !ok {
			return nil, errorsmod.Wrap(ErrInvalidProof, "trie node must be a list")
		}

		switch len(list) {
		case 2: // leaf or extension node
			pathBytes, ok := list[0].([]byte)
			if !ok {
				return nil, errorsmod.Wrap(ErrInvalidProof, "invalid path in node")
			}
			nodePath, isLeaf := hexPrefixDecode(pathBytes)
			remaining := len(nibbles) - idx
			if len(nodePath) > remaining {
				return nil, nil // key diverges: proven absent
			}
			for i, n := range nodePath {
				if nibbles[idx+i] != n {
					return nil, nil // key diverges: proven absent
				}
			}
			idx += len(nodePath)

			if isLeaf {
				if idx != len(nibbles) {
					return nil, nil // stored key is shorter: proven absent
				}
				value, ok := list[1].([]byte)
				if !ok {
					return nil, errorsmod.Wrap(ErrInvalidProof, "invalid leaf value")
				}
				return value, nil
			}

			// extension node: descend into the child
			next, err := resolveChild(nodes, list[1])
			if err != nil {
				return nil, err
			}
			if idx == len(nibbles) {
				// The key is exhausted at an extension node; values only
				// live in leaves, so the key is absent.
				return nil, nil
			}
			node = next

		case 17: // branch node
			if idx == len(nibbles) {
				value, ok := list[16].([]byte)
				if !ok || len(value) == 0 {
					return nil, nil // key proven absent
				}
				return value, nil
			}
			child := list[nibbles[idx]]
			empty, ok := child.([]byte)
			if ok && len(empty) == 0 {
				return nil, nil // no child in this direction: proven absent
			}
			idx++
			next, err := resolveChild(nodes, child)
			if err != nil {
				return nil, err
			}
			node = next

		default:
			return nil, errorsmod.Wrapf(ErrInvalidProof, "invalid trie node list length %d", len(list))
		}
	}
}

// resolveChild resolves a child reference: either a 32-byte keccak hash into
// the proof node set, or an inline RLP node (decoded as a nested list).
func resolveChild(nodes map[common.Hash][]byte, ref interface{}) ([]byte, error) {
	switch child := ref.(type) {
	case []byte:
		if len(child) == 32 {
			if node, ok := nodes[common.BytesToHash(child)]; ok {
				return node, nil
			}
			return nil, errorsmod.Wrap(ErrInvalidProof, "proof is missing a hashed child node")
		}
		if len(child) == 0 {
			return nil, errorsmod.Wrap(ErrInvalidProof, "empty inline child node")
		}
		// Inline nodes must be lists in the parent encoding; a non-empty
		// byte string that is not a hash is invalid.
		return nil, errorsmod.Wrap(ErrInvalidProof, "invalid child node reference")
	case []interface{}:
		encoded, err := rlp.EncodeToBytes(child)
		if err != nil {
			return nil, errorsmod.Wrap(ErrInvalidProof, "failed to re-encode inline child node")
		}
		return encoded, nil
	default:
		return nil, errorsmod.Wrap(ErrInvalidProof, "invalid child node reference")
	}
}

// keyToNibbles splits a hash into nibbles (4-bit values).
func keyToNibbles(key []byte) []byte {
	nibbles := make([]byte, len(key)*2)
	for i, b := range key {
		nibbles[i*2] = b >> 4
		nibbles[i*2+1] = b & 0x0f
	}
	return nibbles
}

// hexPrefixDecode decodes a hex-prefix encoded path, returning the path
// nibbles and whether it terminates a leaf node.
func hexPrefixDecode(hp []byte) (path []byte, isLeaf bool) {
	if len(hp) == 0 {
		return nil, false
	}
	flag := hp[0] >> 4
	odd := flag & 1
	isLeaf = flag&2 != 0

	if odd == 1 {
		path = append(path, hp[0]&0x0f)
	}
	for _, b := range hp[1:] {
		path = append(path, b>>4, b&0x0f)
	}
	return path, isLeaf
}

// verifyAccountStorageRoot proves the storage root of the contract account
// at address from the state root of a finalized header.
func verifyAccountStorageRoot(stateRoot [32]byte, address []byte, proof [][]byte, storageRoot [32]byte) error {
	account, err := getAccount(stateRoot, address, proof)
	if err != nil {
		return err
	}
	if account.StorageRoot != common.BytesToHash(storageRoot[:]) {
		return errorsmod.Wrapf(ErrValueMismatch,
			"proven account storage root %X does not match claimed storage root %X", account.StorageRoot, storageRoot)
	}
	return nil
}

// getAccount returns the account proven at address under stateRoot.
func getAccount(stateRoot [32]byte, address []byte, proof [][]byte) (*ethAccount, error) {
	value, err := trieGet(stateRoot, address, proof)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errorsmod.Wrap(ErrInvalidProof, "account is absent from the state trie")
	}
	var account ethAccount
	if err := rlp.DecodeBytes(value, &account); err != nil {
		return nil, errorsmod.Wrap(ErrInvalidProof, "invalid account RLP encoding")
	}
	return &account, nil
}

// verifyStorageInclusionProof proves that value (as a big-endian 32-byte
// integer) is stored at key in the trie rooted at storageRoot.
func verifyStorageInclusionProof(storageRoot [32]byte, key [32]byte, value [32]byte, proof [][]byte) error {
	stored, err := trieGet(storageRoot, key[:], proof)
	if err != nil {
		return err
	}
	if stored == nil {
		return errorsmod.Wrap(ErrInvalidProof, "storage value is missing")
	}
	expected := rlpEncodeUint(value)
	if !bytes.Equal(stored, expected) {
		return errorsmod.Wrapf(ErrValueMismatch, "stored value %X does not match expected %X", stored, expected)
	}
	return nil
}

// verifyStorageExclusionProof proves that no value is stored at key in the
// trie rooted at storageRoot.
func verifyStorageExclusionProof(storageRoot [32]byte, key [32]byte, proof [][]byte) error {
	stored, err := trieGet(storageRoot, key[:], proof)
	if err != nil {
		return err
	}
	if stored != nil {
		return errorsmod.Wrapf(ErrValueMismatch, "storage value %X should be absent", stored)
	}
	return nil
}

// rlpEncodeUint encodes a 32-byte big-endian integer as a canonical RLP
// integer (minimal big-endian, zero encoded as the empty string).
func rlpEncodeUint(be [32]byte) []byte {
	i := 0
	for i < 32 && be[i] == 0 {
		i++
	}
	minimal := be[i:]
	// rlpEncodeUint never mutates its input; copy when appending the prefix.
	out := make([]byte, 0, len(minimal)+1)
	switch {
	case len(minimal) == 1 && minimal[0] < 0x80:
		return append(out, minimal...)
	case len(minimal) <= 55:
		return append(append(out, 0x80+byte(len(minimal))), minimal...)
	default:
		return append(append(out, 0xb7+byte(len(minimal)-55)), minimal...)
	}
}
