package spectre

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// These tests pin the hardening that the spectre repository's Rust client
// (cw-ics08-wasm-eth) gained on 2026-10-27 (audit item #572), which this module
// must match: anchoring headers at the latest consensus state, refusing
// misbehaviour signed by a past committee, requiring a next sync committee at
// bootstrap, and exact branch lengths.

const parityFixture = "Test_ICS20TransferERC20TokenfromEthereumToCosmosAndBack"

// fixtureClient is a spectre client initialized from a fixture and driven
// through the light client module interface.
type fixtureClient struct {
	t        *testing.T
	module   LightClientModule
	provider clienttypes.StoreProvider
	ctx      sdk.Context
	clientID string
	cs       *ClientState
	headers  []*Header
}

func newFixtureClient(t *testing.T, fixtureName string) *fixtureClient {
	t.Helper()
	cdc := getTestCodec()
	fixture := loadFixture(t, fixtureName)
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	consState := decodeConsensusState(initial["consensus_state"])

	ctx, storeKey := setupTestApp(t, time.Unix(1, 0))
	provider := clienttypes.NewStoreProvider(runtime.NewKVStoreService(storeKey))
	c := &fixtureClient{
		t:        t,
		module:   NewLightClientModule(cdc, provider),
		provider: provider,
		ctx:      ctx,
		clientID: "spectre-0",
		cs:       cs,
	}

	csBz, err := cs.Marshal()
	require.NoError(t, err)
	consBz, err := consState.Marshal()
	require.NoError(t, err)
	require.NoError(t, c.module.Initialize(ctx, c.clientID, csBz, consBz))

	for i := 1; i < len(fixture.Steps); i++ {
		for _, fh := range decodeRelayerMessages(t, fixture.Steps[i].Data).headers {
			c.headers = append(c.headers, fh.toProto())
		}
	}
	return c
}

// ctxAt returns a context whose block time is shortly after the header's
// attested execution timestamp, as the reference tests use.
func (c *fixtureClient) ctxAt(h *Header) sdk.Context {
	ts := h.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000
	return sdk.NewContext(c.ctx.MultiStore(), cmtproto.Header{Time: time.Unix(int64(ts), 0)}, false, log.NewNopLogger())
}

// apply verifies and applies header i through the module interface.
func (c *fixtureClient) apply(i int) {
	c.t.Helper()
	h := c.headers[i]
	require.NoError(c.t, c.module.VerifyClientMessage(c.ctxAt(h), c.clientID, h), "header %d", i)
	require.False(c.t, c.module.CheckForMisbehaviour(c.ctxAt(h), c.clientID, h))
	c.module.UpdateState(c.ctxAt(h), c.clientID, h)
}

func (c *fixtureClient) latestSlot() uint64 {
	return c.module.LatestHeight(c.ctx, c.clientID).GetRevisionHeight()
}

func (c *fixtureClient) consensusState(slot uint64) *ConsensusState {
	store := c.provider.ClientStore(c.ctx, c.clientID)
	cs, _ := GetConsensusState(c.ctx, store, getTestCodec(), clienttypes.NewHeight(0, slot))
	return cs
}

// cloneHeader returns a deep copy of a header, so a case can mutate it.
func cloneHeader(t *testing.T, h *Header) *Header {
	t.Helper()
	bz, err := h.Marshal()
	require.NoError(t, err)
	var out Header
	require.NoError(t, out.Unmarshal(bz))
	return &out
}

func TestHeaderAnchoredAtAStaleSlotIsRejected(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	require.GreaterOrEqual(t, len(c.headers), 2, "the fixture must advance the client at least twice")
	for i := range c.headers {
		c.apply(i)
	}
	latest := c.latestSlot()

	stale := cloneHeader(t, c.headers[0])
	require.Less(t, stale.TrustedSlot, latest, "header 0 is anchored below the client's latest slot")
	finalized := stale.ConsensusUpdate.FinalizedHeader.Beacon.Slot
	before := c.consensusState(finalized)
	require.NotNil(t, before)

	err := c.module.VerifyClientMessage(c.ctxAt(stale), c.clientID, stale)
	require.ErrorIs(t, err, ErrTrustedSlotNotLatest)

	// The apply step does not rely on verification having run.
	func() {
		defer func() {
			r := recover()
			applyErr, ok := r.(error)
			require.True(t, ok, "expected an error panic, got %v", r)
			require.ErrorIs(t, applyErr, ErrTrustedSlotNotLatest)
		}()
		c.module.UpdateState(c.ctxAt(stale), c.clientID, stale)
	}()

	require.Equal(t, latest, c.latestSlot())
	require.Equal(t, before, c.consensusState(finalized), "a stale header leaves the stored consensus state untouched")
}

func TestHeaderAnchoredAtTheLatestSlotStillAdvancesTheClient(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	for i := range c.headers {
		c.apply(i)
	}
	last := c.headers[len(c.headers)-1]
	require.Equal(t, last.ConsensusUpdate.FinalizedHeader.Beacon.Slot, c.latestSlot())
}

// conflictingEvidence builds misbehaviour out of the second fixture header:
// its valid update and a copy attesting a different execution state root.
func conflictingEvidence(t *testing.T, c *fixtureClient) (*Misbehaviour, *Header) {
	t.Helper()
	h := cloneHeader(t, c.headers[1])
	other := cloneHeader(t, c.headers[1])
	other.ConsensusUpdate.AttestedHeader.Execution.StateRoot[0] ^= 0xff
	return &Misbehaviour{
		TrustedSlot:         h.TrustedSlot,
		ActiveSyncCommittee: h.ActiveSyncCommittee,
		IsNextCommittee:     h.IsNextCommittee,
		Update_1:            h.ConsensusUpdate,
		Update_2:            other.ConsensusUpdate,
	}, h
}

// cloneConsensusState returns a deep copy of a consensus state, so a case can
// mutate it.
func cloneConsensusState(t *testing.T, cs *ConsensusState) *ConsensusState {
	t.Helper()
	bz, err := cs.Marshal()
	require.NoError(t, err)
	var out ConsensusState
	require.NoError(t, out.Unmarshal(bz))
	return &out
}

// The signing committee is bound to the signature period (altair/electra
// light client sync protocol): a stored+1 signature must come from the next
// committee and a stored signature from the current one. A committee signing
// outside its tenure must not verify, whatever the header's is_next flag
// claims: otherwise a retiring committee could keep signing updates one
// period past its tenure (and install an attacker-chosen committee as the
// stored next one), or an incoming committee could act one period early.
func TestSigningCommitteeMustMatchTheSignaturePeriod(t *testing.T) {
	c := newFixtureClient(t, parityFixture)

	t.Run("retiring committee signing a stored+1 period is rejected", func(t *testing.T) {
		h := cloneHeader(t, c.headers[0])
		require.True(t, h.IsNextCommittee, "header 0 is signed by the anchor's next committee")
		anchor := cloneConsensusState(t, c.consensusState(c.latestSlot()))
		require.NotNil(t, anchor.NextSyncCommittee, "the anchor knows the next committee")

		// Pretend the signing committee is the stored current one: the anchor
		// now trusts it for the current period, and the header claims a
		// signature from the next period signed by that committee.
		anchor.CurrentSyncCommittee = *anchor.NextSyncCommittee
		h.IsNextCommittee = false

		stored, err := c.cs.computeSyncCommitteePeriodAtSlot(anchor.Slot)
		require.NoError(t, err)
		signature, err := c.cs.computeSyncCommitteePeriodAtSlot(h.ConsensusUpdate.SignatureSlot)
		require.NoError(t, err)
		require.Equal(t, stored+1, signature, "the header signs one period past the anchor")

		err = c.cs.verifyHeader(anchor, uint64(c.ctxAt(h).BlockTime().Unix()), h)
		require.ErrorIs(t, err, ErrInvalidSignaturePeriod)
	})

	t.Run("incoming committee signing the stored period is rejected", func(t *testing.T) {
		c.apply(0) // advance to header 1's anchor
		h := cloneHeader(t, c.headers[1])
		require.False(t, h.IsNextCommittee, "header 1 is signed by the stored current committee")
		anchor := cloneConsensusState(t, c.consensusState(c.latestSlot()))
		require.Equal(t, h.TrustedSlot, anchor.Slot)
		require.NotNil(t, anchor.NextSyncCommittee)

		// Pretend the signing committee is the stored next one: the header
		// claims a same-period signature by a committee that has not taken
		// over yet.
		next := anchor.CurrentSyncCommittee
		anchor.NextSyncCommittee = &next
		h.IsNextCommittee = true

		stored, err := c.cs.computeSyncCommitteePeriodAtSlot(anchor.Slot)
		require.NoError(t, err)
		signature, err := c.cs.computeSyncCommitteePeriodAtSlot(h.ConsensusUpdate.SignatureSlot)
		require.NoError(t, err)
		require.Equal(t, stored, signature, "the header signs within the anchor's period")

		err = c.cs.verifyHeader(anchor, uint64(c.ctxAt(h).BlockTime().Unix()), h)
		require.ErrorIs(t, err, ErrInvalidSignaturePeriod)
	})
}

func TestMisbehaviourSignedByAPastCommitteeIsRejected(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	require.GreaterOrEqual(t, len(c.headers), 2)
	for i := range c.headers {
		c.apply(i)
	}
	mis, h := conflictingEvidence(t, c)
	anchor := c.consensusState(mis.TrustedSlot)
	require.NotNil(t, anchor)
	now := uint64(c.ctxAt(h).BlockTime().Unix())

	cs := c.cs.Clone()
	cs.LatestSlot = c.latestSlot()
	cs.LatestExecutionBlockNumber = 0

	signaturePeriod, err := cs.computeSyncCommitteePeriodAtSlot(mis.Update_1.SignatureSlot)
	require.NoError(t, err)
	latestPeriod, err := cs.computeSyncCommitteePeriodAtSlot(cs.LatestSlot)
	require.NoError(t, err)
	require.Equal(t, latestPeriod, signaturePeriod, "the evidence is signed in the client's current period")

	t.Run("current period passes the committee guard", func(t *testing.T) {
		err := cs.verifyMisbehaviour(anchor, now, mis)
		require.Error(t, err, "the second update is tampered")
		require.NotErrorIs(t, err, ErrSignedByPastCommittee)
	})

	t.Run("client moved on to a later period", func(t *testing.T) {
		later := cs.Clone()
		later.LatestSlot = (signaturePeriod + 1) * later.SlotsPerEpoch * later.EpochsPerSyncCommitteePeriod
		err := later.verifyMisbehaviour(anchor, now, mis)
		require.ErrorIs(t, err, ErrSignedByPastCommittee)
	})

	t.Run("checkForMisbehaviour does not report it", func(t *testing.T) {
		later := cs.Clone()
		later.LatestSlot = (signaturePeriod + 1) * later.SlotsPerEpoch * later.EpochsPerSyncCommitteePeriod
		require.Error(t, later.verifyMisbehaviour(anchor, now, mis))
	})
}

// An anchor from the previous period is still enough for evidence signed by
// the committee that took over from it, which the anchor knows as its next sync
// committee.
func TestMisbehaviourFromAnOlderAnchorSignedByTheCurrentCommitteePassesTheGuard(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	for i := range c.headers {
		c.apply(i)
	}
	first := cloneHeader(t, c.headers[0])
	require.True(t, first.IsNextCommittee, "header 0 is signed by the next committee of its anchor")
	other := cloneHeader(t, c.headers[0])
	other.ConsensusUpdate.AttestedHeader.Execution.StateRoot[0] ^= 0xff
	mis := &Misbehaviour{
		TrustedSlot:         first.TrustedSlot,
		ActiveSyncCommittee: first.ActiveSyncCommittee,
		IsNextCommittee:     true,
		Update_1:            first.ConsensusUpdate,
		Update_2:            other.ConsensusUpdate,
	}

	cs := c.cs.Clone()
	cs.LatestSlot = c.latestSlot()
	anchor := c.consensusState(mis.TrustedSlot)
	require.Less(t, mis.TrustedSlot, cs.LatestSlot, "the anchor is older than the latest state")

	err := cs.verifyMisbehaviour(anchor, uint64(c.ctxAt(first).BlockTime().Unix()), mis)
	require.Error(t, err, "the second update is tampered")
	require.NotErrorIs(t, err, ErrSignedByPastCommittee)
}

func TestInitializeRejectsAnUnusableBootstrap(t *testing.T) {
	fixture := loadFixture(t, parityFixture)
	initial := rawJSON(fixture.Steps[0].Data)

	tests := []struct {
		name    string
		mut     func(cs *ClientState, cons *ConsensusState)
		wantErr error
	}{
		{"no next sync committee", func(_ *ClientState, cons *ConsensusState) { cons.NextSyncCommittee = nil }, clienttypes.ErrInvalidConsensus},
		{"frozen client state", func(cs *ClientState, _ *ConsensusState) { cs.Frozen = true }, ErrInvalidClientState},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := decodeClientState(initial["client_state"])
			cons := decodeConsensusState(initial["consensus_state"])
			tc.mut(cs, cons)

			ctx, storeKey := setupTestApp(t, time.Unix(1, 0))
			module := NewLightClientModule(getTestCodec(), clienttypes.NewStoreProvider(runtime.NewKVStoreService(storeKey)))
			csBz, err := cs.Marshal()
			require.NoError(t, err)
			consBz, err := cons.Marshal()
			require.NoError(t, err)
			require.ErrorIs(t, module.Initialize(ctx, "spectre-0", csBz, consBz), tc.wantErr)
		})
	}
}

func TestUpdateBranchesMustHaveTheExactDepth(t *testing.T) {
	fixture := loadFixture(t, parityFixture)
	runner := newFixtureRunner(t, fixture)
	header := decodeRelayerMessages(t, fixture.Steps[1].Data).headers[0].toProto()
	trusted := runner.consStates[header.TrustedSlot]
	currentSlot, err := runner.clientState.computeSlotAtTimestamp(header.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000)
	require.NoError(t, err)
	require.NotNil(t, header.ConsensusUpdate.NextSyncCommittee, "header 0 carries the next sync committee")

	validate := func(mut func(u *LightClientUpdate)) error {
		h := cloneHeader(t, header)
		mut(&h.ConsensusUpdate)
		return runner.clientState.validateLightClientUpdate(trusted, h.ActiveSyncCommittee, &h.ConsensusUpdate, currentSlot, h.IsNextCommittee)
	}
	require.NoError(t, validate(func(*LightClientUpdate) {}), "the unmutated update verifies")

	zero := make([]byte, 32)
	tests := []struct {
		name string
		mut  func(u *LightClientUpdate)
		want string
	}{
		{"finality branch one short", func(u *LightClientUpdate) { u.FinalityBranch = u.FinalityBranch[1:] }, "finality branch has 6 nodes, expected 7"},
		{"finality branch one long", func(u *LightClientUpdate) {
			u.FinalityBranch = append([][]byte{zero}, u.FinalityBranch...)
		}, "finality branch has 8 nodes, expected 7"},
		{"next committee branch one short", func(u *LightClientUpdate) {
			u.NextSyncCommitteeBranch = u.NextSyncCommitteeBranch[1:]
		}, "next sync committee branch has 5 nodes, expected 6"},
		{"execution branch one short", func(u *LightClientUpdate) {
			u.AttestedHeader.ExecutionBranch = u.AttestedHeader.ExecutionBranch[1:]
		}, "execution branch has 3 nodes, expected 4"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.mut)
			require.ErrorIs(t, err, ErrInvalidMerkleBranch)
			require.ErrorContains(t, err, tc.want)
		})
	}
}
