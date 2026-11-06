package spectre

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// TestFixedExecutionFieldLengthsAreEnforced covers the truncation collision:
// a field one byte longer than its spec length hashes like the honest
// encoding (rightPad truncates), so without exact length checks a header
// stores an unprovable consensus state and a misbehaviour forges a conflict
// out of a single honest signature.
func TestFixedExecutionFieldLengthsAreEnforced(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	runner := newFixtureRunner(t, fixture)
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	header := msgs.headers[0]
	trusted := runner.consStates[header.TrustedSlot]
	currentTime := header.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000
	currentSlot, err := runner.clientState.computeSlotAtTimestamp(currentTime)
	require.NoError(t, err)
	headerRaw := fixtureHeaderRaw(t, fixture, 1)

	// The unmutated update still verifies.
	valid := decodeHeader(headerRaw).toProto()
	require.NoError(t, runner.clientState.validateLightClientUpdate(
		trusted, valid.ActiveSyncCommittee, &valid.ConsensusUpdate, currentSlot, valid.IsNextCommittee))

	appendByte := func(b []byte) []byte {
		return append(append([]byte(nil), b...), 0)
	}
	execFields := []struct {
		name string
		mut  func(e *ExecutionPayloadHeader)
	}{
		{"parent_hash", func(e *ExecutionPayloadHeader) { e.ParentHash = appendByte(e.ParentHash) }},
		{"fee_recipient", func(e *ExecutionPayloadHeader) { e.FeeRecipient = e.FeeRecipient[:19] }},
		{"state_root", func(e *ExecutionPayloadHeader) { e.StateRoot = appendByte(e.StateRoot) }},
		{"receipts_root", func(e *ExecutionPayloadHeader) { e.ReceiptsRoot = appendByte(e.ReceiptsRoot) }},
		{"prev_randao", func(e *ExecutionPayloadHeader) { e.PrevRandao = appendByte(e.PrevRandao) }},
		{"base_fee_per_gas", func(e *ExecutionPayloadHeader) { e.BaseFeePerGas = e.BaseFeePerGas[:31] }},
		{"block_hash", func(e *ExecutionPayloadHeader) { e.BlockHash = appendByte(e.BlockHash) }},
		{"transactions_root", func(e *ExecutionPayloadHeader) { e.TransactionsRoot = appendByte(e.TransactionsRoot) }},
		{"withdrawals_root", func(e *ExecutionPayloadHeader) { e.WithdrawalsRoot = appendByte(e.WithdrawalsRoot) }},
	}
	beaconFields := []struct {
		name string
		mut  func(b *BeaconBlockHeader)
	}{
		{"parent_root", func(b *BeaconBlockHeader) { b.ParentRoot = appendByte(b.ParentRoot) }},
		{"state_root", func(b *BeaconBlockHeader) { b.StateRoot = appendByte(b.StateRoot) }},
		{"body_root", func(b *BeaconBlockHeader) { b.BodyRoot = appendByte(b.BodyRoot) }},
	}

	for _, headerName := range []string{"attested", "finalized"} {
		for _, tc := range execFields {
			t.Run(headerName+" "+tc.name, func(t *testing.T) {
				h := decodeHeader(headerRaw).toProto()
				target := h.ConsensusUpdate.AttestedHeader.Execution
				if headerName == "finalized" {
					target = h.ConsensusUpdate.FinalizedHeader.Execution
				}
				tc.mut(target)
				err := runner.clientState.validateLightClientUpdate(
					trusted, h.ActiveSyncCommittee, &h.ConsensusUpdate, currentSlot, h.IsNextCommittee)
				require.Error(t, err, "the malformed field must be rejected before any hashing")
			})
		}
		for _, tc := range beaconFields {
			t.Run(headerName+" beacon "+tc.name, func(t *testing.T) {
				h := decodeHeader(headerRaw).toProto()
				target := &h.ConsensusUpdate.AttestedHeader.Beacon
				if headerName == "finalized" {
					target = &h.ConsensusUpdate.FinalizedHeader.Beacon
				}
				tc.mut(target)
				err := runner.clientState.validateLightClientUpdate(
					trusted, h.ActiveSyncCommittee, &h.ConsensusUpdate, currentSlot, h.IsNextCommittee)
				require.Error(t, err)
			})
		}
	}

	// A misbehaviour whose update 2 differs from the honest update only by an
	// oversized state root. The bounds check must reject it before anything is
	// hashed: the oversized field truncates to the same SSZ root, so every
	// branch and signature check would otherwise pass.
	h1 := decodeHeader(headerRaw).toProto()
	h2 := decodeHeader(headerRaw).toProto()
	h2.ConsensusUpdate.AttestedHeader.Execution.StateRoot = appendByte(h2.ConsensusUpdate.AttestedHeader.Execution.StateRoot)
	mis := &Misbehaviour{
		TrustedSlot:         h1.TrustedSlot,
		ActiveSyncCommittee: h1.ActiveSyncCommittee,
		IsNextCommittee:     h1.IsNextCommittee,
		Update_1:            h1.ConsensusUpdate,
		Update_2:            h2.ConsensusUpdate,
	}
	anchor := runner.consStates[mis.TrustedSlot]
	require.NotNil(t, anchor)
	err = runner.clientState.verifyMisbehaviour(anchor, currentTime, mis)
	require.Error(t, err, "misbehaviour with an oversized field must be rejected")
}

// TestUpdateStateWithMisbehaviourReturnsNoHeights verifies updateState does
// not panic on a Misbehaviour: the RedundantRelayDecorator calls UpdateState
// during CheckTx for every top-level MsgUpdateClient, with misbehaviour
// checks omitted.
func TestUpdateStateWithMisbehaviourReturnsNoHeights(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	c.apply(0)
	mis, _ := conflictingEvidence(t, c)

	store := c.provider.ClientStore(c.ctx, c.clientID)
	heights := c.cs.updateState(c.ctx, getTestCodec(), store, mis)
	require.Empty(t, heights)
	require.Equal(t, exported.Active, c.module.Status(c.ctx, c.clientID),
		"a misbehaviour passed to updateState must not freeze the client")
}

// TestSyncAggregatePaddingBitsAreRejected verifies the Bitvector semantics:
// bits beyond the configured committee size must be zero, and the
// supermajority counts real members only, so padding cannot inflate a small
// committee past 2/3.
func TestSyncAggregatePaddingBitsAreRejected(t *testing.T) {
	// Committee of 10: two bytes, members 0-9 in the low bits of byte two.
	// Six real signers are not 2/3 of 10 (18 < 20), and counting the padding
	// must not change that.
	sa := &SyncAggregate{SyncCommitteeBits: []byte{0b00111111, 0b00000000}}
	require.Equal(t, uint64(6), sa.numParticipants(10))
	require.False(t, sa.hasSupermajority(10))

	// numParticipants stops at the committee size: members 8 and 9 are the
	// low bits of the second byte; the padding above them does not count.
	sa2 := &SyncAggregate{SyncCommitteeBits: []byte{0x00, 0b00000011}}
	require.Equal(t, uint64(2), sa2.numParticipants(10))
	sa3 := &SyncAggregate{SyncCommitteeBits: []byte{0xFF, 0b00000011}}
	require.Equal(t, uint64(10), sa3.numParticipants(10))

	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	runner := newFixtureRunner(t, fixture)
	headerRaw := fixtureHeaderRaw(t, fixture, 1)

	validate := func(t *testing.T, committeeSize uint64, bits []byte) error {
		t.Helper()
		cs := runner.clientState.Clone()
		cs.SyncCommitteeSize = committeeSize
		cs.MinSyncCommitteeParticipants = 1
		h := decodeHeader(headerRaw).toProto()
		h.ConsensusUpdate.SyncAggregate.SyncCommitteeBits = bits
		return cs.validateLightClientUpdate(runner.consStates[h.TrustedSlot], h.ActiveSyncCommittee, &h.ConsensusUpdate, 1<<40, h.IsNextCommittee)
	}

	t.Run("set padding bits are rejected", func(t *testing.T) {
		// Member 10 is padding for a committee of 10: the low second-byte
		// position is what the wrong shift of the first fix missed.
		err := validate(t, 10, []byte{0xFF, 0b00000100})
		require.ErrorIs(t, err, ErrInvalidSyncCommittee)
		require.ErrorContains(t, err, "padding")

		err = validate(t, 14, []byte{0xFF, 0b01111111})
		require.ErrorContains(t, err, "padding")
	})

	t.Run("an honest all-signed committee is not padding-rejected", func(t *testing.T) {
		// Committee of 14 with all members signed: members 8-13 are the low
		// six bits of the second byte. This must get past the padding check;
		// it fails later at signature verification, which the fixture cannot
		// satisfy for a resized committee.
		err := validate(t, 14, []byte{0xFF, 0b00111111})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "padding")
		require.NotContains(t, err.Error(), "bitfield length")
	})
}
