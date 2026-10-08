package spectre

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestValidateLightClientUpdateBranches exercises each rejection branch of
// validateLightClientUpdate by mutating a valid fixture update.
func TestValidateLightClientUpdateBranches(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	runner := newFixtureRunner(t, fixture)
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	header := msgs.headers[0]
	trusted := runner.consStates[header.TrustedSlot]
	currentTime := header.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000

	currentSlot, err := runner.clientState.computeSlotAtTimestamp(currentTime)
	require.NoError(t, err)

	// headerJSON retains the raw encoding so every case starts from a
	// pristine copy (proto copies share nested slices).
	headerRaw := fixtureHeaderRaw(t, fixture, 1)

	mkUpdate := func(mut func(u *LightClientUpdate, cs *ClientState)) (*LightClientUpdate, *ClientState) {
		h := decodeHeader(headerRaw).toProto()
		cs := runner.clientState
		mut(&h.ConsensusUpdate, cs)
		return &h.ConsensusUpdate, cs
	}

	tests := []struct {
		name string
		mut  func(u *LightClientUpdate, cs *ClientState)
	}{
		{"bitfield wrong size", func(u *LightClientUpdate, cs *ClientState) {
			u.SyncAggregate.SyncCommitteeBits = u.SyncAggregate.SyncCommitteeBits[:len(u.SyncAggregate.SyncCommitteeBits)-1]
		}},
		{"below min participants", func(u *LightClientUpdate, cs *ClientState) {
			u.SyncAggregate.SyncCommitteeBits = make([]byte, len(u.SyncAggregate.SyncCommitteeBits))
		}},
		{"attested execution branch broken", func(u *LightClientUpdate, cs *ClientState) {
			u.AttestedHeader.ExecutionBranch[0][0] ^= 0xff
		}},
		{"attested before electra", func(u *LightClientUpdate, cs *ClientState) {
			u.AttestedHeader.Beacon.Slot = (cs.ForkParameters.Electra.Epoch * cs.SlotsPerEpoch) - 1
		}},
		{"finalized slot is genesis", func(u *LightClientUpdate, cs *ClientState) {
			u.FinalizedHeader.Beacon.Slot = cs.GenesisSlot
		}},
		{"signature slot in the future", func(u *LightClientUpdate, cs *ClientState) {
			u.SignatureSlot = currentSlot + 1
		}},
		{"signature slot not after attested", func(u *LightClientUpdate, cs *ClientState) {
			u.SignatureSlot = u.AttestedHeader.Beacon.Slot
		}},
		{"attested before finalized", func(u *LightClientUpdate, cs *ClientState) {
			u.FinalizedHeader.Beacon.Slot = u.AttestedHeader.Beacon.Slot + 1
		}},
		{"signature period skips", func(u *LightClientUpdate, cs *ClientState) {
			u.SignatureSlot += cs.SlotsPerEpoch * cs.EpochsPerSyncCommitteePeriod * 2
			u.AttestedHeader.Beacon.Slot = u.SignatureSlot - 1
			u.FinalizedHeader.Beacon.Slot = u.SignatureSlot - 1
		}},
		{"irrelevant update", func(u *LightClientUpdate, cs *ClientState) {
			u.AttestedHeader.Beacon.Slot = trusted.Slot - 1
			u.FinalizedHeader.Beacon.Slot = trusted.Slot - 1
		}},
		{"finalized execution branch broken", func(u *LightClientUpdate, cs *ClientState) {
			u.FinalizedHeader.ExecutionBranch[0][0] ^= 0xff
		}},
		{"finality branch broken", func(u *LightClientUpdate, cs *ClientState) {
			u.FinalityBranch[0][0] ^= 0xff
		}},
		{"next committee without branch", func(u *LightClientUpdate, cs *ClientState) {
			u.NextSyncCommittee = header.ConsensusUpdate.NextSyncCommittee
			u.NextSyncCommitteeBranch = nil
		}},
		{"next committee wrong size", func(u *LightClientUpdate, cs *ClientState) {
			u.NextSyncCommitteeBranch = make([][]byte, floorLog2(nextSyncCommitteeGindex))
			u.NextSyncCommittee = &SyncCommittee{Pubkeys: make([][]byte, 3), AggregatePubkey: make([]byte, 48)}
			for i := range u.NextSyncCommitteeBranch {
				u.NextSyncCommitteeBranch[i] = make([]byte, 32)
			}
		}},
		{"next committee branch broken", func(u *LightClientUpdate, cs *ClientState) {
			u.NextSyncCommittee = header.ConsensusUpdate.NextSyncCommittee
			u.NextSyncCommitteeBranch = header.ConsensusUpdate.NextSyncCommitteeBranch
			if u.NextSyncCommitteeBranch != nil {
				u.NextSyncCommitteeBranch[0][0] ^= 0xff
			}
		}},
		{"unexpected next committee", func(u *LightClientUpdate, cs *ClientState) {
			u.NextSyncCommittee = &SyncCommittee{Pubkeys: make([][]byte, int(cs.SyncCommitteeSize)), AggregatePubkey: make([]byte, 48)}
		}},
		{"corrupted signature", func(u *LightClientUpdate, cs *ClientState) {
			u.SyncAggregate.SyncCommitteeSignature[0] ^= 0xff
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			update, cs := mkUpdate(tc.mut)
			err := cs.validateLightClientUpdate(
				trusted, header.ActiveSyncCommittee, update, currentSlot,
			)
			require.Error(t, err, "expected rejection")
		})
	}

	// The unmutated update verifies.
	validUpdate, validCS := mkUpdate(func(u *LightClientUpdate, cs *ClientState) {})
	require.NoError(t, validCS.validateLightClientUpdate(
		trusted, header.ActiveSyncCommittee, validUpdate, currentSlot,
	))
}

// mkLeafRaw builds an RLP leaf node under a key as raw bytes suitable for
// embedding with rlp.RawValue.
func mkLeafRaw(t *testing.T, key []byte, value []byte) []byte {
	t.Helper()
	return rlpEncodeListItems(t, [][]byte{hexPrefix(key, true), value})
}

// mkLeaf builds an RLP leaf node under a key.
func mkLeaf(t *testing.T, key []byte, value []byte) []byte {
	t.Helper()
	return mkLeafRaw(t, key, value)
}

// mkBranchRaw builds an RLP branch node from raw (already encoded) items.
func mkBranchRaw(t *testing.T, items []interface{}) []byte {
	t.Helper()
	encoded, err := rlp.EncodeToBytes(items)
	require.NoError(t, err)
	return encoded
}

// mkBranch builds an RLP branch node from 16 children and an optional value.
func mkBranch(t *testing.T, children [][]byte, value []byte) []byte {
	t.Helper()
	var items []interface{}
	for _, child := range children {
		items = append(items, child)
	}
	items = append(items, value)
	return mkBranchRaw(t, items)
}

// mkExt builds an RLP extension node whose child is embedded inline.
func mkExt(t *testing.T, key []byte, child []byte) []byte {
	t.Helper()
	encoded, err := rlp.EncodeToBytes([]interface{}{hexPrefix(key, false), rlp.RawValue(child)})
	require.NoError(t, err)
	return encoded
}

// hexPrefix encodes a key path with the hex-prefix encoding.
func hexPrefix(key []byte, terminator bool) []byte {
	flag := byte(0)
	if terminator {
		flag = 2
	}
	if len(key)%2 == 1 {
		flag |= 1
		hp := []byte{flag<<4 | key[0]}
		for i := 1; i < len(key); i += 2 {
			hp = append(hp, key[i]<<4|key[i+1])
		}
		return hp
	}
	hp := []byte{flag << 4}
	for i := 0; i < len(key); i += 2 {
		hp = append(hp, key[i]<<4|key[i+1])
	}
	return hp
}

// rlpEncodeList encodes items as an RLP list.
func rlpEncodeList(t *testing.T, items ...[]byte) []byte {
	t.Helper()
	return rlpEncodeListItems(t, items)
}

// rlpEncodeListItems encodes a full item slice as an RLP list.
func rlpEncodeListItems(t *testing.T, items [][]byte) []byte {
	t.Helper()
	var asInterfaces []interface{}
	for _, item := range items {
		asInterfaces = append(asInterfaces, item)
	}
	encoded, err := rlp.EncodeToBytes(asInterfaces)
	require.NoError(t, err)
	return encoded
}

// TestTrieGetWalks exercises the MPT walk with synthetic tries: inclusion,
// exclusion at leaves, branches and extensions, inline children, and
// malformed inputs.
func TestTrieGetWalks(t *testing.T) {
	key := []byte("a")
	nibbles := keyToNibbles(crypto.Keccak256(key))

	t.Run("leaf inclusion", func(t *testing.T) {
		leaf := mkLeaf(t, nibbles, []byte("value"))
		root := keccakTo32(crypto.Keccak256Hash(leaf))
		val, err := trieGet(root, key, [][]byte{leaf})
		require.NoError(t, err)
		require.Equal(t, []byte("value"), val)
	})

	t.Run("leaf exclusion diverging path", func(t *testing.T) {
		diverged := append([]byte(nil), nibbles...)
		diverged[len(diverged)-1] ^= 1
		otherLeaf := mkLeaf(t, diverged, []byte("other"))
		root := keccakTo32(crypto.Keccak256Hash(otherLeaf))
		val, err := trieGet(root, key, [][]byte{otherLeaf})
		require.NoError(t, err)
		require.Nil(t, val)
	})

	t.Run("leaf exclusion shorter path", func(t *testing.T) {
		short := nibbles[:len(nibbles)-1]
		otherLeaf := mkLeaf(t, short, []byte("short"))
		root := keccakTo32(crypto.Keccak256Hash(otherLeaf))
		val, err := trieGet(root, key, [][]byte{otherLeaf})
		require.NoError(t, err)
		require.Nil(t, val, "stored key shorter than the queried key proves absence")
	})

	t.Run("branch inclusion and exclusion", func(t *testing.T) {
		child := mkLeaf(t, nibbles[1:], []byte("deep"))
		childHash := crypto.Keccak256Hash(child)
		children := make([][]byte, 16)
		children[nibbles[0]] = childHash[:]
		branch := mkBranch(t, children, nil)
		root := keccakTo32(crypto.Keccak256Hash(branch))

		val, err := trieGet(root, key, [][]byte{branch, child})
		require.NoError(t, err)
		require.Equal(t, []byte("deep"), val)

		// A key whose first nibble is empty proves absence.
		other := append([]byte(nil), key...)
		other[0] ^= 0xff
		val, err = trieGet(root, other, [][]byte{branch})
		require.NoError(t, err)
		require.Nil(t, val)
	})

	t.Run("branch value at exhausted key", func(t *testing.T) {
		// A branch with only an empty value slot at the queried direction
		// proves absence.
		children := make([][]byte, 16)
		branch := mkBranch(t, children, nil)
		root := keccakTo32(crypto.Keccak256Hash(branch))
		val, err := trieGet(root, key, [][]byte{branch})
		require.NoError(t, err)
		require.Nil(t, val, "empty branch slot proves absence")
	})

	t.Run("extension inclusion and exclusion", func(t *testing.T) {
		// ext over the first nibble, then a branch, then a leaf.
		deepChild := mkLeaf(t, nibbles[2:], []byte("deeper"))
		branchChildren := make([][]byte, 16)
		deepChildHash := crypto.Keccak256Hash(deepChild)
		branchChildren[nibbles[1]] = deepChildHash[:]
		branch := mkBranch(t, branchChildren, nil)
		ext := mkExt(t, nibbles[:1], branch)
		root := keccakTo32(crypto.Keccak256Hash(ext))

		val, err := trieGet(root, key, [][]byte{ext, branch, deepChild})
		require.NoError(t, err)
		require.Equal(t, []byte("deeper"), val)

		// A diverging key at the extension path proves absence.
		other := append([]byte(nil), key...)
		other[0] ^= 0xff
		val, err = trieGet(root, other, [][]byte{ext})
		require.NoError(t, err)
		require.Nil(t, val)
	})

	t.Run("inline child node", func(t *testing.T) {
		// A branch whose child is an inline leaf (RLP under 32 bytes) that
		// diverges from the queried key.
		inlineLeaf := mkLeafRaw(t, nibbles[1:5], []byte{0x01})
		if len(inlineLeaf) >= 32 {
			t.Skip("synthetic inline node too large")
		}
		items := make([]interface{}, 17)
		for i := range items {
			items[i] = []byte{}
		}
		items[nibbles[0]] = rlp.RawValue(inlineLeaf)
		branch := mkBranchRaw(t, items)
		root := keccakTo32(crypto.Keccak256Hash(branch))

		// The queried key diverges after the inline leaf path.
		val, err := trieGet(root, key, [][]byte{branch})
		require.NoError(t, err)
		require.Nil(t, val, "inline leaf with a diverging path proves absence")
	})

	t.Run("malformed inputs", func(t *testing.T) {
		leaf := mkLeaf(t, nibbles, []byte("value"))
		root := keccakTo32(crypto.Keccak256Hash(leaf))

		// A missing root node errors.
		_, err := trieGet([32]byte{}, key, [][]byte{leaf})
		require.Error(t, err)

		// Garbage node encodings error.
		garbageRoot := keccakTo32(crypto.Keccak256Hash([]byte("garbage")))
		_, err = trieGet(garbageRoot, key, [][]byte{[]byte("garbage")})
		require.Error(t, err)

		// Empty proof nodes error.
		_, err = trieGet(root, key, [][]byte{nil})
		require.Error(t, err)

		// A scalar RLP value as root errors.
		scalar, err := rlp.EncodeToBytes("scalar")
		require.NoError(t, err)
		_, err = trieGet(keccakTo32(crypto.Keccak256Hash(scalar)), key, [][]byte{scalar})
		require.Error(t, err)

		// A wrong-sized node list errors.
		bad := mkBranchRaw(t, []interface{}{[]byte{}, []byte{}, []byte{}, []byte{}, []byte{}})
		_, err = trieGet(keccakTo32(crypto.Keccak256Hash(bad)), key, [][]byte{bad})
		require.Error(t, err)
	})
}

// keccakTo32 converts a common.Hash to a fixed array.
func keccakTo32(h common.Hash) [32]byte {
	var out [32]byte
	copy(out[:], h[:])
	return out
}
