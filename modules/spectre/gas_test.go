package spectre

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// meteredAt returns a context like c.ctxAt(h) with a finite gas meter, plus
// the meter to read consumption from.
func meteredAt(c *fixtureClient, h *Header, limit uint64) (sdk.Context, types.GasMeter) {
	ctx := c.ctxAt(h).WithGasMeter(types.NewGasMeter(limit))
	return ctx, ctx.GasMeter()
}

// TestGasCostFormula verifies the charged cost scales with the committee
// size and the number of BLS passes, and saturates instead of overflowing.
func TestGasCostFormula(t *testing.T) {
	for _, size := range []uint64{32, 512} {
		cs := &ClientState{SyncCommitteeSize: size}
		perPass := GasCostBlsPass + GasCostPerPubkey*size
		require.Equal(t, 2*perPass, cs.GasCostHeaderVerification(), "header: one aggregation, one verification, size %d", size)
		require.Equal(t, 3*perPass, cs.GasCostMisbehaviourVerification(), "misbehaviour: one aggregation, two verifications, size %d", size)
	}

	// An absurd committee size saturates instead of wrapping to cheap gas.
	huge := &ClientState{SyncCommitteeSize: math.MaxUint64}
	require.Equal(t, uint64(math.MaxUint64), huge.GasCostHeaderVerification())
	require.Equal(t, uint64(math.MaxUint64), huge.GasCostMisbehaviourVerification())
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
		assertVerificationCharged(t, meter, c.cs.GasCostHeaderVerification())
	})

	t.Run("invalid header pays the header cost in full", func(t *testing.T) {
		h := cloneHeader(t, c.headers[0])
		h.ConsensusUpdate.SyncAggregate.SyncCommitteeSignature[0] ^= 0xff
		ctx, meter := meteredAt(c, h, math.MaxUint64)
		require.Error(t, c.module.VerifyClientMessage(ctx, c.clientID, h))
		assertVerificationCharged(t, meter, c.cs.GasCostHeaderVerification())
	})

	t.Run("misbehaviour pays the misbehaviour cost in full", func(t *testing.T) {
		for i := range c.headers {
			c.apply(i)
		}
		mis, h := conflictingEvidence(t, c)
		ctx, meter := meteredAt(c, h, math.MaxUint64)
		require.Error(t, c.module.VerifyClientMessage(ctx, c.clientID, mis), "the second update is tampered")
		assertVerificationCharged(t, meter, c.cs.GasCostMisbehaviourVerification())
	})
}

// TestVerificationGasExhaustionPanics verifies a meter below the charge
// fails the message with the standard out-of-gas panic.
func TestVerificationGasExhaustionPanics(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	h := cloneHeader(t, c.headers[0])
	ctx, _ := meteredAt(c, h, c.cs.GasCostHeaderVerification()-1)
	require.Panics(t, func() {
		_ = c.module.VerifyClientMessage(ctx, c.clientID, h)
	})
}
