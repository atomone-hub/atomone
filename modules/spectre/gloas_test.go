package spectre

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// These tests mirror the Gloas tests of the spectre repository's Rust light
// client (sync_protocol_helpers.rs gloas_test, update.rs, execution.rs).

const (
	testElectraEpoch = 10
	testGloasEpoch   = 20
	testSlotsPerEp   = 32
)

// gloasClientState returns a client state whose Gloas fork starts at
// gloasEpoch (use ^uint64(0) for unscheduled).
func gloasClientState(gloasEpoch uint64) *ClientState {
	v := func(n byte) []byte { return []byte{n, 0, 0, 0} }
	return &ClientState{
		SlotsPerEpoch: testSlotsPerEp,
		ForkParameters: ForkParameters{
			GenesisForkVersion: v(0),
			Altair:             Fork{Version: v(1)},
			Bellatrix:          Fork{Version: v(2)},
			Capella:            Fork{Version: v(3)},
			Deneb:              Fork{Version: v(4)},
			Electra:            Fork{Version: v(5), Epoch: testElectraEpoch},
			Fulu:               Fork{Version: v(6), Epoch: ^uint64(0)},
			Gloas:              Fork{Version: v(7), Epoch: gloasEpoch},
		},
	}
}

const (
	preGloasSlot = uint64(testElectraEpoch+1) * testSlotsPerEp
	gloasSlot    = uint64(testGloasEpoch) * testSlotsPerEp
)

// testBranch returns n deterministic nodes.
func testBranch(n int, seed byte) [][32]byte {
	out := make([][32]byte, n)
	for i := range out {
		out[i] = sha256Hash([32]byte{byte(i), seed}, [32]byte{seed})
	}
	return out
}

// branchBytes converts nodes to the slice form headers carry.
func branchBytes(nodes [][32]byte) [][]byte {
	out := make([][]byte, len(nodes))
	for i := range nodes {
		out[i] = append([]byte(nil), nodes[i][:]...)
	}
	return out
}

// rootFromBranch folds leaf up the branch following the subtree index of
// gindex, producing the root the branch proves against.
func rootFromBranch(leaf [32]byte, branch [][32]byte, gindex uint64) [32]byte {
	index := subtreeIndex(gindex)
	value := leaf
	for i, node := range branch {
		if index>>uint(i)&1 == 1 {
			value = sha256Hash(node, value)
		} else {
			value = sha256Hash(value, node)
		}
	}
	return value
}

// preGloasHeader builds a pre-Gloas-format light client header at slot whose
// body root commits to its execution payload header.
func preGloasHeader(slot uint64) *LightClientHeader {
	exec := &ExecutionPayloadHeader{
		ParentHash:   make([]byte, 32),
		FeeRecipient: make([]byte, 20),
		StateRoot:    bytes32Of(0x11),
		LogsBloom:    make([]byte, 256),
		BlockNumber:  7,
		Timestamp:    1_000,
		BlockHash:    bytes32Of(0x22),
	}
	branch := testBranch(floorLog2(executionPayloadGindex), 1)
	body := rootFromBranch(executionPayloadHeaderRoot(exec), branch, executionPayloadGindex)
	return &LightClientHeader{
		Beacon:          BeaconBlockHeader{Slot: slot, BodyRoot: body[:]},
		Execution:       exec,
		ExecutionBranch: branchBytes(branch),
	}
}

// gloasHeader builds a Gloas-format light client header at slot, proving the
// block hash at the generalized index of the slot's own fork. Headers that
// predate Gloas carry the Gloas-length branch with leading zero nodes.
func gloasHeader(cs *ClientState, slot uint64) *LightClientHeader {
	blockHash := [32]byte{0xAA}
	gindex := uint64(executionBlockHashGindexDeneb)
	if cs.isGloasEpoch(cs.computeEpochAtSlot(slot)) {
		gindex = executionBlockHashGindexGloas
	}
	real := testBranch(floorLog2(gindex), 2)
	branch := make([][32]byte, floorLog2(executionBlockHashGindexGloas)-len(real))
	branch = append(branch, real...)
	body := rootFromBranch(blockHash, real, gindex)
	return &LightClientHeader{
		Beacon:             BeaconBlockHeader{Slot: slot, BodyRoot: body[:]},
		ExecutionBlockHash: blockHash[:],
		ExecutionBranch:    branchBytes(branch),
	}
}

func bytes32Of(b byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestGindicesSwitchAtTheGloasEpoch(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)

	g, err := cs.finalizedRootGindexAtSlot(gloasSlot - 1)
	require.NoError(t, err)
	require.EqualValues(t, finalizedRootGindexElectra, g)
	g, err = cs.finalizedRootGindexAtSlot(gloasSlot)
	require.NoError(t, err)
	require.EqualValues(t, finalizedRootGindexGloas, g)

	g, err = cs.nextSyncCommitteeGindexAtSlot(gloasSlot - 1)
	require.NoError(t, err)
	require.EqualValues(t, nextSyncCommitteeGindexElectra, g)
	g, err = cs.nextSyncCommitteeGindexAtSlot(gloasSlot)
	require.NoError(t, err)
	require.EqualValues(t, nextSyncCommitteeGindexGloas, g)

	// A slot before electra is not supported at all.
	_, err = cs.finalizedRootGindexAtSlot(uint64(testElectraEpoch-1) * testSlotsPerEp)
	require.ErrorIs(t, err, ErrMustBeElectraOrLater)
}

func TestGindicesStayOnElectraWhenGloasIsUnscheduled(t *testing.T) {
	cs := gloasClientState(^uint64(0))
	g, err := cs.finalizedRootGindexAtSlot(gloasSlot * 1000)
	require.NoError(t, err)
	require.EqualValues(t, finalizedRootGindexElectra, g)
	g, err = cs.nextSyncCommitteeGindexAtSlot(gloasSlot * 1000)
	require.NoError(t, err)
	require.EqualValues(t, nextSyncCommitteeGindexElectra, g)
}

func TestGloasGindexDepths(t *testing.T) {
	require.Equal(t, 7, floorLog2(finalizedRootGindexElectra))
	require.Equal(t, 6, floorLog2(nextSyncCommitteeGindexElectra))
	require.Equal(t, 9, floorLog2(finalizedRootGindexGloas))
	require.Equal(t, 11, floorLog2(nextSyncCommitteeGindexGloas))
	require.Equal(t, 9, floorLog2(executionBlockHashGindexDeneb))
	require.Equal(t, 11, floorLog2(executionBlockHashGindexGloas))
}

func TestPreGloasHeaderIsAcceptedBeforeTheGloasEpoch(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	require.NoError(t, cs.isValidLightClientHeader(preGloasHeader(preGloasSlot)))
}

func TestExecutionPayloadHeaderIsRejectedFromTheGloasEpoch(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	err := cs.isValidLightClientHeader(preGloasHeader(gloasSlot))
	require.ErrorIs(t, err, ErrInvalidHeader)
	require.ErrorContains(t, err, "gloas epoch")
}

func TestGloasHeaderWithValidBlockHashProofIsAccepted(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	require.NoError(t, cs.isValidLightClientHeader(gloasHeader(cs, gloasSlot)))
}

func TestGloasFormatHeaderBeforeTheGloasEpochUsesAPaddedDenebProof(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	require.NoError(t, cs.isValidLightClientHeader(gloasHeader(cs, preGloasSlot)),
		"a normalized block hash proof is accepted before the gloas epoch")
}

func TestGloasHeaderWithWrongBlockHashIsRejected(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	header := gloasHeader(cs, gloasSlot)
	header.ExecutionBlockHash[0] ^= 0xff
	require.ErrorIs(t, cs.isValidLightClientHeader(header), ErrInvalidMerkleBranch)
}

func TestGloasEpochHeaderCannotReuseADenebDepthProof(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	header := gloasHeader(cs, gloasSlot)
	// The Deneb-depth proof of the same block hash, without the leading zero
	// padding, has the wrong length for a Gloas-format header.
	header.ExecutionBranch = header.ExecutionBranch[floorLog2(executionBlockHashGindexGloas)-floorLog2(executionBlockHashGindexDeneb):]
	err := cs.isValidLightClientHeader(header)
	require.ErrorIs(t, err, ErrInvalidMerkleBranch)
	require.ErrorContains(t, err, "expected 11")
}

func TestPreGloasEpochRejectsNonzeroPaddingInTheNormalizedBranch(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	header := gloasHeader(cs, preGloasSlot)
	header.ExecutionBranch[0][0] = 1
	require.ErrorIs(t, cs.isValidLightClientHeader(header), ErrInvalidMerkleBranch)
}

func TestBlockHashBranchMustHaveTheGloasLength(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)
	for _, n := range []int{0, 10, 12} {
		header := gloasHeader(cs, gloasSlot)
		header.ExecutionBranch = branchBytes(testBranch(n, 3))
		err := cs.isValidLightClientHeader(header)
		require.ErrorIs(t, err, ErrInvalidMerkleBranch, "branch of %d nodes", n)
	}
}

func TestLightClientHeaderMustSetExactlyOneExecutionCommitment(t *testing.T) {
	cs := gloasClientState(testGloasEpoch)

	both := preGloasHeader(preGloasSlot)
	both.ExecutionBlockHash = bytes32Of(0xAA)
	require.ErrorIs(t, cs.isValidLightClientHeader(both), ErrInvalidHeader)
	require.Error(t, validateLightClientHeaderBasic(both, "test"))

	neither := preGloasHeader(preGloasSlot)
	neither.Execution = nil
	require.ErrorIs(t, cs.isValidLightClientHeader(neither), ErrInvalidHeader)
	require.Error(t, validateLightClientHeaderBasic(neither, "test"))

	short := gloasHeader(cs, gloasSlot)
	short.ExecutionBlockHash = short.ExecutionBlockHash[:31]
	require.Error(t, validateLightClientHeaderBasic(short, "test"))
}

// testExecutionHeader holds the fields a test asserts on.
type testExecutionHeader struct {
	stateRoot []byte
	number    uint64
	timestamp uint64
}

// executionHeaderRLP encodes an execution block header with the item layout of
// a post-Prague header (twenty-one items) and returns it with its block hash.
// The light client reads only the state root (index 3), number (8) and
// timestamp (11).
func executionHeaderRLP(t *testing.T) ([]byte, common.Hash, testExecutionHeader) {
	t.Helper()
	want := testExecutionHeader{
		stateRoot: common.HexToHash("0x3344").Bytes(),
		number:    256,
		timestamp: 1_700_000_123,
	}
	items := []any{
		common.HexToHash("0x0102").Bytes(),    // 0  parent hash
		common.HexToHash("0x1dcc").Bytes(),    // 1  uncle hash
		common.HexToAddress("0xdead").Bytes(), // 2  coinbase
		want.stateRoot,                        // 3  state root
		common.HexToHash("0x56e8").Bytes(),    // 4  transactions root
		common.HexToHash("0x56e9").Bytes(),    // 5  receipts root
		make([]byte, 256),                     // 6  logs bloom
		uint64(0),                             // 7  difficulty
		want.number,                           // 8  number
		uint64(30_000_000),                    // 9  gas limit
		uint64(21_000),                        // 10 gas used
		want.timestamp,                        // 11 timestamp
		[]byte("spectre"),                     // 12 extra data
		common.HexToHash("0x77").Bytes(),      // 13 mix hash
		make([]byte, 8),                       // 14 nonce
		uint64(7),                             // 15 base fee
		common.HexToHash("0x56ea").Bytes(),    // 16 withdrawals root
		uint64(0),                             // 17 blob gas used
		uint64(0),                             // 18 excess blob gas
		common.HexToHash("0x01").Bytes(),      // 19 parent beacon root
		common.HexToHash("0xe3b0").Bytes(),    // 20 requests hash
	}
	raw, err := rlp.EncodeToBytes(items)
	require.NoError(t, err)
	return raw, crypto.Keccak256Hash(raw), want
}

func TestVerifyExecutionHeaderRLP(t *testing.T) {
	raw, hash, want := executionHeaderRLP(t)

	summary, err := verifyExecutionHeaderRLP(raw, hash)
	require.NoError(t, err)
	require.Equal(t, want.stateRoot, summary.stateRoot)
	require.EqualValues(t, want.number, summary.blockNumber)
	require.EqualValues(t, want.timestamp, summary.timestamp)

	t.Run("hash mismatch", func(t *testing.T) {
		wrong := hash
		wrong[0] ^= 0xff
		_, err := verifyExecutionHeaderRLP(raw, wrong)
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "hashes to")
	})

	t.Run("oversized", func(t *testing.T) {
		big := make([]byte, maxExecutionHeaderRLPBytes+1)
		_, err := verifyExecutionHeaderRLP(big, crypto.Keccak256Hash(big))
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "maximum")
	})

	t.Run("not a list", func(t *testing.T) {
		scalar, err := rlp.EncodeToBytes([]byte("scalar"))
		require.NoError(t, err)
		_, err = verifyExecutionHeaderRLP(scalar, crypto.Keccak256Hash(scalar))
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
	})

	t.Run("trailing bytes", func(t *testing.T) {
		trailing := append(append([]byte(nil), raw...), 0x00)
		_, err := verifyExecutionHeaderRLP(trailing, crypto.Keccak256Hash(trailing))
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "trailing")
	})

	t.Run("too few items", func(t *testing.T) {
		short, err := rlp.EncodeToBytes([]any{[]byte{1}, []byte{2}, []byte{3}})
		require.NoError(t, err)
		_, err = verifyExecutionHeaderRLP(short, crypto.Keccak256Hash(short))
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
	})

	t.Run("nested list item", func(t *testing.T) {
		items := make([]any, 12)
		for i := range items {
			items[i] = []byte{0x01}
		}
		items[executionHeaderStateRootIndex] = make([]byte, 32)
		items[5] = []any{[]byte{0x01}}
		nested, err := rlp.EncodeToBytes(items)
		require.NoError(t, err)
		_, err = verifyExecutionHeaderRLP(nested, crypto.Keccak256Hash(nested))
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "is a list")
	})

	t.Run("state root of the wrong size", func(t *testing.T) {
		items := make([]any, 12)
		for i := range items {
			items[i] = []byte{0x01}
		}
		bad, err := rlp.EncodeToBytes(items)
		require.NoError(t, err)
		_, err = verifyExecutionHeaderRLP(bad, crypto.Keccak256Hash(bad))
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "state root")
	})

	t.Run("non-canonical block number", func(t *testing.T) {
		items := make([]any, 12)
		for i := range items {
			items[i] = []byte{0x01}
		}
		items[executionHeaderStateRootIndex] = make([]byte, 32)
		items[executionHeaderNumberIndex] = []byte{0x00, 0x05} // leading zero
		bad, err := rlp.EncodeToBytes(items)
		require.NoError(t, err)
		_, err = verifyExecutionHeaderRLP(bad, crypto.Keccak256Hash(bad))
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "block number")
	})
}

func TestFinalizedExecutionSummary(t *testing.T) {
	raw, hash, want := executionHeaderRLP(t)

	pre := &Header{ConsensusUpdate: LightClientUpdate{FinalizedHeader: *preGloasHeader(preGloasSlot)}}
	summary, err := finalizedExecutionSummary(pre)
	require.NoError(t, err)
	require.Equal(t, bytes32Of(0x11), summary.stateRoot)
	require.EqualValues(t, 7, summary.blockNumber)
	require.EqualValues(t, 1_000, summary.timestamp)

	gloas := &Header{
		ConsensusUpdate:    LightClientUpdate{FinalizedHeader: LightClientHeader{ExecutionBlockHash: hash[:]}},
		ExecutionHeaderRlp: raw,
	}
	summary, err = finalizedExecutionSummary(gloas)
	require.NoError(t, err)
	require.Equal(t, want.stateRoot, summary.stateRoot)
	require.EqualValues(t, want.number, summary.blockNumber)

	t.Run("gloas header rejects a state root the block hash does not commit to", func(t *testing.T) {
		wrong := &Header{
			ConsensusUpdate:    LightClientUpdate{FinalizedHeader: LightClientHeader{ExecutionBlockHash: bytes32Of(0x99)}},
			ExecutionHeaderRlp: raw,
		}
		_, err := finalizedExecutionSummary(wrong)
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
	})

	t.Run("gloas header requires the rlp", func(t *testing.T) {
		missing := &Header{ConsensusUpdate: LightClientUpdate{FinalizedHeader: LightClientHeader{ExecutionBlockHash: hash[:]}}}
		_, err := finalizedExecutionSummary(missing)
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "required")
	})

	t.Run("pre-gloas header rejects a stray rlp", func(t *testing.T) {
		stray := &Header{
			ConsensusUpdate:    LightClientUpdate{FinalizedHeader: *preGloasHeader(preGloasSlot)},
			ExecutionHeaderRlp: raw,
		}
		_, err := finalizedExecutionSummary(stray)
		require.ErrorIs(t, err, ErrInvalidExecutionHeader)
		require.ErrorContains(t, err, "pre-Gloas")
	})

	t.Run("neither commitment", func(t *testing.T) {
		_, err := finalizedExecutionSummary(&Header{})
		require.ErrorIs(t, err, ErrInvalidHeader)
	})
}

func TestConflictingExecutionRoots(t *testing.T) {
	a := preGloasHeader(preGloasSlot)
	b := preGloasHeader(preGloasSlot)
	b.Execution.StateRoot = bytes32Of(0x77)

	r1, r2, err := conflictingExecutionRoots(a, b)
	require.NoError(t, err)
	require.NotEqual(t, r1, r2, "two payload headers are compared by state root")

	// Equal state roots with different block hashes are not a conflict in
	// the pre-Gloas format.
	c := preGloasHeader(preGloasSlot)
	c.Execution.BlockHash = bytes32Of(0x55)
	r1, r2, err = conflictingExecutionRoots(a, c)
	require.NoError(t, err)
	require.Equal(t, r1, r2)

	// As soon as either header is in the Gloas format both are compared by
	// execution block hash.
	cs := gloasClientState(testGloasEpoch)
	g := gloasHeader(cs, gloasSlot)
	r1, r2, err = conflictingExecutionRoots(a, g)
	require.NoError(t, err)
	require.Equal(t, bytes32(a.Execution.BlockHash), r1)
	require.Equal(t, bytes32(g.ExecutionBlockHash), r2)

	_, _, err = conflictingExecutionRoots(a, &LightClientHeader{})
	require.ErrorIs(t, err, ErrInvalidHeader)
}

func TestHeaderValidateBasicBoundsTheExecutionHeaderRLP(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	header := decodeRelayerMessages(t, fixture.Steps[1].Data).headers[0].toProto()
	require.NoError(t, header.ValidateBasic())

	header.ExecutionHeaderRlp = make([]byte, maxExecutionHeaderRLPBytes+1)
	require.ErrorIs(t, header.ValidateBasic(), ErrInvalidHeader)
}
