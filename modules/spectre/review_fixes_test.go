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

	// The misbehaviour shape from the review: update 2 differs from the honest
	// update only by an oversized state root. It truncates to the same SSZ
	// root, so every branch and signature check passes; the length check must
	// be what rejects it, before the zero-substituted root comparison counts
	// it as a conflict.
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
	heights := c.cs.updateState(getTestCodec(), store, mis)
	require.Empty(t, heights)
	require.Equal(t, exported.Active, c.module.Status(c.ctx, c.clientID),
		"a misbehaviour passed to updateState must not freeze the client")
}

// TestSyncAggregatePaddingBitsAreRejected verifies the Bitvector semantics:
// bits beyond the configured committee size must be zero, and the
// supermajority counts real members only, so padding cannot inflate a small
// committee past 2/3.
func TestSyncAggregatePaddingBitsAreRejected(t *testing.T) {
	// Committee of 10: one byte plus 6 padding bits. Six real signers are 2/3
	// of 10? 6*3 = 18 >= 20 is false, so 6 alone fails; adding set padding
	// bits must not change that.
	bits := []byte{0b00111111} // 6 real signers, 2 padding bits clear
	sa := &SyncAggregate{SyncCommitteeBits: bits}
	require.Equal(t, uint64(6), sa.numParticipants(10))
	require.False(t, sa.hasSupermajority(10))

	// Same real signers with the two padding bits set: the aggregate is now a
	// malformed bitvector and must be rejected outright by the update check.
	padded := &SyncAggregate{SyncCommitteeBits: []byte{0b00111111 | 0b11000000}}

	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	runner := newFixtureRunner(t, fixture)
	headerRaw := fixtureHeaderRaw(t, fixture, 1)

	// A client with a committee of 10: the bitfield length must be 2 bytes.
	small := runner.clientState.Clone()
	small.SyncCommitteeSize = 10
	small.MinSyncCommitteeParticipants = 1
	padded.SyncCommitteeBits = []byte{0xFF, 0b01000000} // length 2, padding set
	h := decodeHeader(headerRaw).toProto()
	err := small.validateLightClientUpdate(runner.consStates[h.TrustedSlot], h.ActiveSyncCommittee, &h.ConsensusUpdate, 1<<40, h.IsNextCommittee)
	require.Error(t, err, "set padding bits must be rejected")

	// numParticipants stops at the committee size: members 8 and 9 are the low
	// bits of the second byte; the 6 padding bits above them do not count.
	sa2 := &SyncAggregate{SyncCommitteeBits: []byte{0x00, 0b00000011}}
	require.Equal(t, uint64(2), sa2.numParticipants(10))
	sa3 := &SyncAggregate{SyncCommitteeBits: []byte{0xFF, 0b00000011}}
	require.Equal(t, uint64(10), sa3.numParticipants(10))
}
