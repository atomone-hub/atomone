package spectre

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"

	"cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// meteredAt returns a context like c.ctxAt(h) with a finite gas meter, plus
// the meter to read consumption from.
func meteredAt(c *fixtureClient, h *Header, limit uint64) (sdk.Context, types.GasMeter) {
	ctx := c.ctxAt(h).WithGasMeter(types.NewGasMeter(limit))
	return ctx, ctx.GasMeter()
}

// bitsAllSet returns a bitfield of length ceil(size/8) with the first size
// bits set, and bitsAllClear the same length with none set.
func bitsAllSet(size uint64) []byte   { return filledBits(size, 0xFF) }
func bitsAllClear(size uint64) []byte { return filledBits(size, 0x00) }

func filledBits(size uint64, fill byte) []byte {
	bits := make([]byte, (size+7)/8)
	for i := range bits {
		bits[i] = fill
	}
	if pad := size % 8; pad != 0 && len(bits) > 0 {
		bits[len(bits)-1] &= (1 << pad) - 1
	}
	return bits
}

// TestGasCostFormula verifies the charge scales with the signature checks and
// the non-signers whose keys the verifier decodes, and saturates instead of
// overflowing.
func TestGasCostFormula(t *testing.T) {
	for _, size := range []uint64{32, 512} {
		cs := &ClientState{SyncCommitteeSize: size}

		// Full participation: only the fixed check is charged.
		require.Equal(t, GasCostBlsCheck, cs.GasCostHeaderVerification(bitsAllSet(size)), "all signing, size %d", size)

		// No participation: every key of the committee is decoded.
		require.Equal(t, GasCostBlsCheck+GasCostPerPubkey*size, cs.GasCostHeaderVerification(bitsAllClear(size)), "none signing, size %d", size)

		// A misbehaviour is charged for both updates' signature checks and
		// both updates' non-signers.
		require.Equal(t, 2*GasCostBlsCheck+2*GasCostPerPubkey*size,
			cs.GasCostMisbehaviourVerification(bitsAllClear(size), bitsAllClear(size)), "misbehaviour, none signing, size %d", size)
	}

	// An absurd committee size saturates instead of wrapping to cheap gas.
	huge := &ClientState{SyncCommitteeSize: math.MaxUint64}
	require.Equal(t, uint64(math.MaxUint64), huge.GasCostHeaderVerification(bitsAllClear(math.MaxUint64)))
	require.Equal(t, uint64(math.MaxUint64), huge.GasCostMisbehaviourVerification(nil, nil))
}

// meteredStoreReadOverhead bounds the gas the verification path consumes
// beyond the charged cost: the client and consensus state reads through the
// store provider are gas-metered too (~3.5k gas on the fixtures).
const meteredStoreReadOverhead uint64 = 20_000

// assertVerificationCharged checks the verification cost was paid in full,
// and that consumption beyond it is only metered store reads.
func assertVerificationCharged(t *testing.T, meter types.GasMeter, cost uint64) {
	t.Helper()
	consumed := meter.GasConsumed()
	require.GreaterOrEqual(t, consumed, cost, "the verification charge must be paid in full")
	require.LessOrEqual(t, consumed, cost+meteredStoreReadOverhead,
		"consumption beyond the charge is limited to metered store reads")
}

// TestVerificationGasChargedBeforeBlsWork verifies the charge lands before
// any BLS work: a message that fails at the signature check still pays in
// full, so garbage-signature spam cannot reach the expensive part for free.
func TestVerificationGasChargedBeforeBlsWork(t *testing.T) {
	c := newFixtureClient(t, parityFixture)

	t.Run("valid header pays the header cost", func(t *testing.T) {
		h := cloneHeader(t, c.headers[0])
		ctx, meter := meteredAt(c, h, math.MaxUint64)
		require.NoError(t, c.module.VerifyClientMessage(ctx, c.clientID, h))
		assertVerificationCharged(t, meter, c.cs.GasCostHeaderVerification(h.ConsensusUpdate.SyncAggregate.SyncCommitteeBits))
	})

	t.Run("invalid header pays the header cost in full", func(t *testing.T) {
		h := cloneHeader(t, c.headers[0])
		h.ConsensusUpdate.SyncAggregate.SyncCommitteeSignature[0] ^= 0xff
		ctx, meter := meteredAt(c, h, math.MaxUint64)
		require.Error(t, c.module.VerifyClientMessage(ctx, c.clientID, h))
		assertVerificationCharged(t, meter, c.cs.GasCostHeaderVerification(h.ConsensusUpdate.SyncAggregate.SyncCommitteeBits))
	})

	t.Run("misbehaviour pays the misbehaviour cost in full", func(t *testing.T) {
		for i := range c.headers {
			c.apply(i)
		}
		mis, h := conflictingEvidence(t, c)
		ctx, meter := meteredAt(c, h, math.MaxUint64)
		require.Error(t, c.module.VerifyClientMessage(ctx, c.clientID, mis), "the second update is tampered")
		assertVerificationCharged(t, meter, c.cs.GasCostMisbehaviourVerification(
			mis.Update_1.SyncAggregate.SyncCommitteeBits, mis.Update_2.SyncAggregate.SyncCommitteeBits))
	})
}

// TestVerificationGasExhaustionPanics verifies a meter below the charge
// fails the message with the standard out-of-gas panic.
func TestVerificationGasExhaustionPanics(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	h := cloneHeader(t, c.headers[0])
	cost := c.cs.GasCostHeaderVerification(h.ConsensusUpdate.SyncAggregate.SyncCommitteeBits)
	ctx, _ := meteredAt(c, h, cost-1)
	require.Panics(t, func() {
		_ = c.module.VerifyClientMessage(ctx, c.clientID, h)
	})
}

// TestFastAggregateVerifyExcluding covers the identity rejections of the
// exclusion verifier and its equivalence with adding the signers.
func TestFastAggregateVerifyExcluding(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	committee := msgs.headers[0].ActiveSyncCommittee
	require.NotNil(t, committee)

	t.Run("a cancelling committee aggregate is rejected", func(t *testing.T) {
		// Subtracting the whole committee from the aggregate leaves the
		// identity, which the standard's key validation rejects.
		err := fastAggregateVerifyExcluding(committee.AggregatePubkey, committee.Pubkeys, [32]byte{}, make([]byte, 96))
		require.ErrorIs(t, err, ErrBLSVerificationFailed)
		require.ErrorContains(t, err, "identity")
	})

	t.Run("an identity committee aggregate is rejected", func(t *testing.T) {
		// The compressed encoding of the identity point parses fine, so the
		// explicit identity rejection is what stops it.
		var inf bls12381.G1Affine
		inf.SetInfinity()
		infBytes := inf.Bytes()
		err := fastAggregateVerifyExcluding(infBytes[:], nil, [32]byte{}, make([]byte, 96))
		require.ErrorContains(t, err, "identity")
	})

	t.Run("equivalence with adding the signers", func(t *testing.T) {
		// The fixture committee's aggregate is the sum of its keys, so
		// subtracting a member then adding it back must reproduce the same
		// aggregate; the equivalence with the signer-sum check itself is
		// exercised end to end by every fixture-driven update test.
		start, err := parsePubkey(committee.AggregatePubkey)
		require.NoError(t, err)
		one, err := parsePubkey(committee.Pubkeys[0])
		require.NoError(t, err)

		var agg bls12381.G1Jac
		agg.FromAffine(&start)
		var neg bls12381.G1Affine
		neg.Neg(&one)
		agg.AddMixed(&neg)
		neg.Neg(&neg)
		agg.AddMixed(&neg)
		var round bls12381.G1Affine
		round.FromJacobian(&agg)
		require.Equal(t, start, round)
	})
}
