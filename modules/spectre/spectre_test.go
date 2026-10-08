package spectre

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	proto "github.com/cosmos/gogoproto/proto"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
)

// makeTestCodec builds a codec that can marshal/unmarshal the spectre types.
func makeTestCodec() codec.Codec {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

// relayerMessages holds the decoded IBC messages of a fixture relayer tx.
type relayerMessages struct {
	headers []*fixtureHeader
	recv    []*channeltypesv2.MsgRecvPacket
	ack     []*channeltypesv2.MsgAcknowledgement
	timeout []*channeltypesv2.MsgTimeout
}

// decodeRelayerMessages decodes the hex TxBody of a fixture step into the
// typed IBC messages. The update client messages carry 08-wasm
// ClientMessages whose data field holds the JSON header; the JSON is decoded
// and the inner header returned.
func decodeRelayerMessages(t *testing.T, data []byte) *relayerMessages {
	t.Helper()
	m := rawJSON(data)
	txHex := fixtureString(m, "relayer_tx_body")
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		t.Fatalf("decode relayer tx body: %v", err)
	}

	// Parse cosmos.tx.v1beta1.TxBody: repeated Any messages = field 1. A raw
	// wire-format unmarshal avoids needing the full interface registry.
	var body tx.TxBody
	if err := proto.Unmarshal(txBytes, &body); err != nil {
		t.Fatalf("unmarshal tx body: %v", err)
	}

	msgs := &relayerMessages{}
	for _, any := range body.Messages {
		switch any.TypeUrl {
		case "/ibc.core.client.v1.MsgUpdateClient":
			var uc clienttypes.MsgUpdateClient
			if err := proto.Unmarshal(any.Value, &uc); err != nil {
				t.Fatalf("unmarshal MsgUpdateClient: %v", err)
			}
			headerJSON, err := wasmClientMessageData(uc.ClientMessage.Value)
			if err != nil {
				t.Fatalf("extract wasm client message data: %v", err)
			}
			msgs.headers = append(msgs.headers, decodeHeader(headerJSON))
		case "/ibc.core.channel.v2.MsgRecvPacket":
			var rp channeltypesv2.MsgRecvPacket
			if err := proto.Unmarshal(any.Value, &rp); err != nil {
				t.Fatalf("unmarshal MsgRecvPacket: %v", err)
			}
			msgs.recv = append(msgs.recv, &rp)
		case "/ibc.core.channel.v2.MsgAcknowledgement":
			var ak channeltypesv2.MsgAcknowledgement
			if err := proto.Unmarshal(any.Value, &ak); err != nil {
				t.Fatalf("unmarshal MsgAcknowledgement: %v", err)
			}
			msgs.ack = append(msgs.ack, &ak)
		case "/ibc.core.channel.v2.MsgTimeout":
			var to channeltypesv2.MsgTimeout
			if err := proto.Unmarshal(any.Value, &to); err != nil {
				t.Fatalf("unmarshal MsgTimeout: %v", err)
			}
			msgs.timeout = append(msgs.timeout, &to)
		default:
			t.Fatalf("unexpected message type %s", any.TypeUrl)
		}
	}
	return msgs
}

// wasmClientMessageData extracts the data field of an
// ibc.lightclients.wasm.v1.ClientMessage, which is a single bytes field.
func wasmClientMessageData(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("empty client message")
	}
	// field 1, wire type 2: tag byte 0x0a followed by varint length
	if value[0] != 0x0a {
		return nil, fmt.Errorf("unexpected wasm client message tag %#x", value[0])
	}
	length, n := varint(value[1:])
	if n == 0 || 1+uint64(n)+length > uint64(len(value)) {
		return nil, fmt.Errorf("malformed wasm client message length")
	}
	return value[1+uint64(n) : 1+uint64(n)+length], nil
}

// varint decodes a protobuf varint, returning the value and bytes consumed.
func varint(b []byte) (uint64, int) {
	var v uint64
	var shift uint
	for i, x := range b {
		v |= uint64(x&0x7f) << shift
		if x&0x80 == 0 {
			return v, i + 1
		}
		shift += 7
		if shift > 63 {
			return 0, 0
		}
	}
	return 0, 0
}

// fixtureRunner drives a fixture through the client lifecycle: initialize,
// verify+apply updates, then verify packet proofs.
type fixtureRunner struct {
	t           *testing.T
	clientState *ClientState
	consStates  map[uint64]*ConsensusState
}

// newFixtureRunner initializes the runner from a fixture's step 0.
func newFixtureRunner(t *testing.T, fixture *stepsFixture) *fixtureRunner {
	t.Helper()
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	consState := decodeConsensusState(initial["consensus_state"])
	if err := cs.Validate(); err != nil {
		t.Fatalf("initial client state invalid: %v", err)
	}
	if err := consState.ValidateBasic(); err != nil {
		t.Fatalf("initial consensus state invalid: %v", err)
	}
	if cs.LatestSlot != consState.Slot {
		t.Fatalf("client latest slot %d != consensus slot %d", cs.LatestSlot, consState.Slot)
	}
	return &fixtureRunner{
		t:           t,
		clientState: cs,
		consStates:  map[uint64]*ConsensusState{consState.Slot: consState},
	}
}

// applyUpdates verifies and applies every header in the relayer messages,
// tracking consensus states per slot like the on-chain client store.
func (r *fixtureRunner) applyUpdates(msgs *relayerMessages) {
	r.t.Helper()
	for i, header := range msgs.headers {
		if err := header.toProto().ValidateBasic(); err != nil {
			r.t.Fatalf("header %d: ValidateBasic: %v", i, err)
		}
		trusted, ok := r.consStates[header.TrustedSlot]
		if !ok {
			r.t.Fatalf("header %d: no consensus state at trusted slot %d", i, header.TrustedSlot)
		}

		// The reference tests verify with a timestamp safely after the
		// update; use the attested execution timestamp + 1000s.
		currentTime := header.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000

		if err := r.clientState.verifyHeader(trusted, currentTime, header.toProto()); err != nil {
			r.t.Fatalf("header %d: verification failed: %v", i, err)
		}

		// Updates apply against the consensus state at the latest slot.
		latest, ok := r.consStates[r.clientState.LatestSlot]
		if !ok {
			r.t.Fatalf("header %d: no consensus state at latest slot %d", i, r.clientState.LatestSlot)
		}
		newCons, newClient, err := r.clientState.updateConsensusState(latest, header.toProto())
		if err != nil {
			r.t.Fatalf("header %d: update failed: %v", i, err)
		}
		r.consStates[newCons.Slot] = newCons
		if newClient != nil {
			r.clientState = newClient
		}
	}
}

// applyAllUpdates applies the headers of every relayer step in the fixture.
func (r *fixtureRunner) applyAllUpdates(fixture *stepsFixture) *relayerMessages {
	r.t.Helper()
	var last *relayerMessages
	for i := 1; i < len(fixture.Steps); i++ {
		msgs := decodeRelayerMessages(r.t, fixture.Steps[i].Data)
		last = msgs
		if len(msgs.headers) > 0 {
			r.applyUpdates(msgs)
		}
	}
	return last
}

// TestFixtureUpdatePipeline verifies and applies every header of every
// fixture: this exercises fork math, merkle branches, tree hashing, BLS
// aggregation checks and signature verification against real testnet data.
func TestFixtureUpdatePipeline(t *testing.T) {
	for _, name := range []string{
		"Test_ICS20TransferERC20TokenfromEthereumToCosmosAndBack",
		"Test_ICS20TransferNativeCosmosCoinsToEthereumAndBack",
		"Test_TimeoutPacketFromCosmos",
	} {
		t.Run(name, func(t *testing.T) {
			fixture := loadFixture(t, name)
			if fixture.Steps == nil {
				t.Skipf("fixture %s is empty", name)
			}
			runner := newFixtureRunner(t, fixture)
			msgs := runner.applyAllUpdates(fixture)
			if len(msgs.headers) == 0 {
				t.Fatalf("fixture contained no update client messages")
			}
		})
	}
}

// TestFixtureMembershipProofs verifies the recv-packet membership proofs of
// the ICS20 fixture against the final consensus state.
func TestFixtureMembershipProofs(t *testing.T) {
	fixture := loadFixture(t, "Test_ICS20TransferERC20TokenfromEthereumToCosmosAndBack")
	runner := newFixtureRunner(t, fixture)

	var recv []*channeltypesv2.MsgRecvPacket
	for i := 1; i < len(fixture.Steps); i++ {
		msgs := decodeRelayerMessages(t, fixture.Steps[i].Data)
		if len(msgs.headers) > 0 {
			runner.applyUpdates(msgs)
		}
		if len(msgs.recv) > 0 {
			recv = append(recv, msgs.recv...)
		}
	}
	if len(recv) == 0 {
		t.Fatal("fixture contained no recv packets")
	}

	for i, rp := range recv {
		proof := decodeMembershipProof(rp.ProofCommitment)
		if proof == nil {
			t.Fatalf("recv %d: invalid membership proof JSON", i)
		}
		consState, ok := runner.consStates[rp.ProofHeight.GetRevisionHeight()]
		if !ok {
			t.Fatalf("recv %d: no consensus state at proof height %d", i, rp.ProofHeight.GetRevisionHeight())
		}

		path := [][]byte{hostv2.PacketCommitmentKey(rp.Packet.SourceClient, rp.Packet.Sequence)}
		value := channeltypesv2.CommitPacket(rp.Packet)

		if err := verifyStorageMembership(runner.clientState, consState, proof, path, value); err != nil {
			t.Fatalf("recv %d: membership verification failed: %v", i, err)
		}
	}
}

// TestFixtureNonMembershipProofs verifies the timeout non-membership proofs
// of the timeout fixture.
func TestFixtureNonMembershipProofs(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	runner := newFixtureRunner(t, fixture)

	var timeouts []*channeltypesv2.MsgTimeout
	for i := 1; i < len(fixture.Steps); i++ {
		msgs := decodeRelayerMessages(t, fixture.Steps[i].Data)
		if len(msgs.headers) > 0 {
			runner.applyUpdates(msgs)
		}
		if len(msgs.timeout) > 0 {
			timeouts = append(timeouts, msgs.timeout...)
		}
	}
	if len(timeouts) == 0 {
		t.Fatal("fixture contained no timeout packets")
	}

	for i, to := range timeouts {
		proof := decodeMembershipProof(to.ProofUnreceived)
		if proof == nil {
			t.Fatalf("timeout %d: invalid membership proof JSON", i)
		}
		consState, ok := runner.consStates[to.ProofHeight.GetRevisionHeight()]
		if !ok {
			t.Fatalf("timeout %d: no consensus state at proof height %d", i, to.ProofHeight.GetRevisionHeight())
		}

		path := [][]byte{hostv2.PacketReceiptKey(to.Packet.DestinationClient, to.Packet.Sequence)}

		if err := verifyStorageNonMembership(runner.clientState, consState, proof, path); err != nil {
			t.Fatalf("timeout %d: non-membership verification failed: %v", i, err)
		}
	}
}
