package spectre

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// setupClientStore sets up a client store for testing.
func setupClientStore(t *testing.T) storetypes.KVStore {
	t.Helper()

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("spectre-test")
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	return ms.GetKVStore(storeKey)
}

// validFixtureClientState returns a client state validated against the
// fixture data.
func validFixtureClientState(t *testing.T) *ClientState {
	t.Helper()
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	require.NoError(t, cs.Validate())
	return cs
}

// TestClientStateValidate exercises every validation branch.
func TestClientStateValidate(t *testing.T) {
	tests := []struct {
		name string
		mut  func(cs *ClientState)
	}{
		{"zero chain id", func(cs *ClientState) { cs.ChainId = 0 }},
		{"short genesis validators root", func(cs *ClientState) { cs.GenesisValidatorsRoot = make([]byte, 31) }},
		{"zero seconds per slot", func(cs *ClientState) { cs.SecondsPerSlot = 0 }},
		{"zero slots per epoch", func(cs *ClientState) { cs.SlotsPerEpoch = 0 }},
		{"zero epochs per period", func(cs *ClientState) { cs.EpochsPerSyncCommitteePeriod = 0 }},
		{"zero committee size", func(cs *ClientState) { cs.SyncCommitteeSize = 0 }},
		{"zero min participants", func(cs *ClientState) { cs.MinSyncCommitteeParticipants = 0 }},
		{"min participants above size", func(cs *ClientState) { cs.MinSyncCommitteeParticipants = cs.SyncCommitteeSize + 1 }},
		{"zero latest slot", func(cs *ClientState) { cs.LatestSlot = 0 }},
		{"latest slot before genesis", func(cs *ClientState) { cs.GenesisSlot = cs.LatestSlot + 1 }},
		{"short contract address", func(cs *ClientState) { cs.IbcContractAddress = make([]byte, 19) }},
		{"short commitment slot", func(cs *ClientState) { cs.IbcCommitmentSlot = make([]byte, 31) }},
		{"short fork version", func(cs *ClientState) { cs.ForkParameters.Electra.Version = make([]byte, 3) }},
		{"decreasing fork epochs", func(cs *ClientState) {
			cs.ForkParameters.Capella.Epoch = cs.ForkParameters.Deneb.Epoch + 1
		}},
		{"fulu scheduled before electra", func(cs *ClientState) {
			cs.ForkParameters.Fulu.Epoch = cs.ForkParameters.Electra.Epoch
		}},
		{"latest slot before electra", func(cs *ClientState) {
			cs.ForkParameters.Electra.Epoch = cs.computeEpochAtSlot(cs.LatestSlot) + 1
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := validFixtureClientState(t)
			tc.mut(cs)
			require.Error(t, cs.Validate())
		})
	}

	// The unmutated fixture client state validates.
	require.NoError(t, validFixtureClientState(t).Validate())
}

// TestConsensusStateValidateBasic exercises consensus state validation.
func TestConsensusStateValidateBasic(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	consState := decodeConsensusState(rawJSON(fixture.Steps[0].Data)["consensus_state"])
	require.NoError(t, consState.ValidateBasic())

	tests := []struct {
		name string
		mut  func(cs *ConsensusState)
	}{
		{"zero slot", func(cs *ConsensusState) { cs.Slot = 0 }},
		{"short state root", func(cs *ConsensusState) { cs.StateRoot = make([]byte, 31) }},
		{"zero timestamp", func(cs *ConsensusState) { cs.Timestamp = 0 }},
		{"short current committee hash", func(cs *ConsensusState) {
			cs.CurrentSyncCommittee.PubkeysHash = make([]byte, 31)
		}},
		{"short next committee aggregate", func(cs *ConsensusState) {
			cs.NextSyncCommittee.AggregatePubkey = make([]byte, 47)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := decodeConsensusState(rawJSON(fixture.Steps[0].Data)["consensus_state"])
			tc.mut(cs)
			require.Error(t, cs.ValidateBasic())
		})
	}
}

// TestHeaderValidateBasic exercises header structural validation.
func TestHeaderValidateBasic(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	header := msgs.headers[0].toProto()
	require.NoError(t, header.ValidateBasic())

	tests := []struct {
		name string
		mut  func(h *Header)
	}{
		{"zero trusted slot", func(h *Header) { h.TrustedSlot = 0 }},
		{"missing committee", func(h *Header) { h.ActiveSyncCommittee = nil }},
		{"short committee pubkey", func(h *Header) {
			h.ActiveSyncCommittee.Pubkeys[0] = make([]byte, 47)
		}},
		{"short aggregate", func(h *Header) {
			h.ActiveSyncCommittee.AggregatePubkey = make([]byte, 47)
		}},
		{"zero signature slot", func(h *Header) { h.ConsensusUpdate.SignatureSlot = 0 }},
		{"short signature", func(h *Header) {
			h.ConsensusUpdate.SyncAggregate.SyncCommitteeSignature = make([]byte, 95)
		}},
		{"empty bits", func(h *Header) { h.ConsensusUpdate.SyncAggregate.SyncCommitteeBits = nil }},
		{"oversized finality branch", func(h *Header) {
			h.ConsensusUpdate.FinalityBranch = make([][]byte, 8)
		}},
		{"finality branch node size", func(h *Header) {
			h.ConsensusUpdate.FinalityBranch[0] = make([]byte, 31)
		}},
		{"committee without branch", func(h *Header) {
			h.ConsensusUpdate.NextSyncCommittee = &SyncCommittee{Pubkeys: make([][]byte, 1)}
			h.ConsensusUpdate.NextSyncCommitteeBranch = nil
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := msgs.headers[0].toProto()
			tc.mut(h)
			require.Error(t, h.ValidateBasic())
		})
	}
}

// TestVerifyMembershipEntryPoint drives verifyMembership and
// verifyNonMembership through the client state methods, including their
// error paths.
func TestVerifyMembershipEntryPoint(t *testing.T) {
	cdc := getTestCodec()

	fixture := loadFixture(t, "Test_ICS20TransferERC20TokenfromEthereumToCosmosAndBack")
	runner := newFixtureRunner(t, fixture)

	var recv *channeltypesv2.MsgRecvPacket
	for i := 1; i < len(fixture.Steps); i++ {
		msgs := decodeRelayerMessages(t, fixture.Steps[i].Data)
		if len(msgs.headers) > 0 {
			runner.applyUpdates(msgs)
		}
		if recv == nil && len(msgs.recv) > 0 {
			recv = msgs.recv[0]
		}
	}
	require.NotNil(t, recv)

	// Materialize the tracked consensus states into a client store.
	clientStore := setupClientStore(t)
	cs := runner.clientState
	for slot, cons := range runner.consStates {
		setConsensusState(clientStore, cdc, cons, clienttypes.NewHeight(0, slot))
	}
	setClientState(clientStore, cdc, cs)

	proofBz, err := cdc.Marshal(decodeMembershipProof(recv.ProofCommitment))
	require.NoError(t, err)
	path := commitmenttypesv2.MerklePath{KeyPath: [][]byte{hostv2.PacketCommitmentKey(recv.Packet.SourceClient, recv.Packet.Sequence)}}
	value := channeltypesv2.CommitPacket(recv.Packet)
	ctx := sdk.Context{} // only block time is read; the methods under test do not use it

	// The valid proof verifies through the entry point.
	require.NoError(t, cs.verifyMembership(ctx, clientStore, cdc, recv.ProofHeight, 0, 0, proofBz, path, value))

	// Frozen clients reject proofs.
	frozen := cs.Clone()
	frozen.Frozen = true
	require.Error(t, frozen.verifyMembership(ctx, clientStore, cdc, recv.ProofHeight, 0, 0, proofBz, path, value))
	require.Error(t, frozen.verifyNonMembership(ctx, clientStore, cdc, recv.ProofHeight, 0, 0, proofBz, path))

	// Non-zero delay periods are rejected.
	require.Error(t, cs.verifyMembership(ctx, clientStore, cdc, recv.ProofHeight, 1, 0, proofBz, path, value))
	require.Error(t, cs.verifyNonMembership(ctx, clientStore, cdc, recv.ProofHeight, 0, 1, proofBz, path))

	// Invalid heights are rejected.
	revHeight := clienttypes.NewHeight(1, recv.ProofHeight.GetRevisionHeight())
	require.Error(t, cs.verifyMembership(ctx, clientStore, cdc, revHeight, 0, 0, proofBz, path, value))
	zeroHeight := clienttypes.NewHeight(0, 0)
	require.Error(t, cs.verifyNonMembership(ctx, clientStore, cdc, zeroHeight, 0, 0, proofBz, path))

	// Heights beyond the latest slot are rejected.
	futureHeight := clienttypes.NewHeight(0, cs.LatestSlot+1)
	require.Error(t, cs.verifyMembership(ctx, clientStore, cdc, futureHeight, 0, 0, proofBz, path, value))
	require.Error(t, cs.verifyNonMembership(ctx, clientStore, cdc, futureHeight, 0, 0, proofBz, path))

	// Missing consensus states are rejected.
	missingHeight := clienttypes.NewHeight(0, recv.ProofHeight.GetRevisionHeight()-1)
	require.Error(t, cs.verifyMembership(ctx, clientStore, cdc, missingHeight, 0, 0, proofBz, path, value))

	// Garbage proofs are rejected.
	require.Error(t, cs.verifyMembership(ctx, clientStore, cdc, recv.ProofHeight, 0, 0, []byte("garbage"), path, value))

	// Wrong path types are rejected.
	require.Error(t, cs.verifyMembership(ctx, clientStore, cdc, recv.ProofHeight, 0, 0, proofBz, nil, value))
	require.Error(t, cs.verifyNonMembership(ctx, clientStore, cdc, recv.ProofHeight, 0, 0, proofBz, nil))
}

// TestVerifyMisbehaviourPaths exercises the misbehaviour verification error
// paths; real conflicting updates do not exist in the fixtures.
func TestVerifyMisbehaviourPaths(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	runner := newFixtureRunner(t, fixture)
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	header := msgs.headers[0]
	trusted := runner.consStates[header.TrustedSlot]
	currentTime := header.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000

	mkMis := func(mut func(mis *Misbehaviour)) *Misbehaviour {
		mis := &Misbehaviour{
			TrustedSlot:         header.TrustedSlot,
			ActiveSyncCommittee: header.ActiveSyncCommittee,
			IsNextCommittee:     header.IsNextCommittee,
			Update_1:            header.ConsensusUpdate,
			Update_2:            header.ConsensusUpdate,
		}
		mut(mis)
		return mis
	}

	// Conflicting state roots are required.
	require.Error(t, runner.clientState.verifyMisbehaviour(trusted, currentTime, mkMis(func(m *Misbehaviour) {})),
		"identical updates are not a conflict")

	// Attested slots must match.
	require.Error(t, runner.clientState.verifyMisbehaviour(trusted, currentTime, mkMis(func(m *Misbehaviour) {
		m.Update_2.AttestedHeader.Beacon.Slot = m.Update_1.AttestedHeader.Beacon.Slot + 1
	})))

	// A tampered second update fails verification.
	require.Error(t, runner.clientState.verifyMisbehaviour(trusted, currentTime, mkMis(func(m *Misbehaviour) {
		m.Update_2.AttestedHeader.Execution.StateRoot[0] ^= 0xff
	})))

	// A tampered committee fails the trusted committee check.
	require.Error(t, runner.clientState.verifyMisbehaviour(trusted, currentTime, mkMis(func(m *Misbehaviour) {
		m.ActiveSyncCommittee.AggregatePubkey[0] ^= 0xff
	})))

	// Structurally invalid misbehaviour is rejected by ValidateBasic.
	require.Error(t, mkMis(func(m *Misbehaviour) { m.TrustedSlot = 0 }).ValidateBasic())
	require.Equal(t, ModuleName, mkMis(func(m *Misbehaviour) {}).ClientType())
}

// TestMisbehaviourFreezeAndStatus verifies freezing through the module
// interface.
func TestMisbehaviourFreezeAndStatus(t *testing.T) {
	cdc := getTestCodec()
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	consState := decodeConsensusState(initial["consensus_state"])

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("spectre-test")
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Time: time.Unix(1, 0)}, false, log.NewNopLogger())

	module := NewLightClientModule(cdc, clienttypes.NewStoreProvider(runtime.NewKVStoreService(storeKey)))
	const clientID = "spectre-0"

	clientStateBz, err := cs.Marshal()
	require.NoError(t, err)
	consensusStateBz, err := consState.Marshal()
	require.NoError(t, err)
	require.NoError(t, module.Initialize(ctx, clientID, clientStateBz, consensusStateBz))

	// Freezing through misbehaviour handling flips the status to frozen.
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	header := msgs.headers[0].toProto()
	module.UpdateStateOnMisbehaviour(ctx, clientID, header)
	require.Equal(t, exported.Frozen, module.Status(ctx, clientID))

	// Frozen clients reject updates at verification.
	updateCtx := sdk.NewContext(ms, cmtproto.Header{
		Time: time.Unix(int64(header.ConsensusUpdate.AttestedHeader.Execution.Timestamp+1000), 0),
	}, false, log.NewNopLogger())
	require.Error(t, module.VerifyClientMessage(updateCtx, clientID, header))

	// Misbehaviour reports on a frozen client report no misbehaviour.
	require.False(t, module.CheckForMisbehaviour(updateCtx, clientID, header))

	// Unknown clients report unknown status.
	require.Equal(t, exported.Unknown, module.Status(ctx, "spectre-999"))

	// Recovery and upgrades are unsupported.
	require.Error(t, module.RecoverClient(ctx, clientID, clientID))
	require.Error(t, module.VerifyUpgradeAndUpdateState(ctx, clientID, nil, nil, nil, nil))
}
