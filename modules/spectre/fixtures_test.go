package spectre

// This file loads the JSON test fixtures produced by the spectre e2e tests
// (Kurtosis ethereum-package + simapp) and converts them into the spectre
// protobuf types. The fixtures use hex strings for byte fields and decimal
// strings for some uint64 fields, matching the cw-ics08-wasm-eth JSON
// encoding.

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/tx"
	proto "github.com/cosmos/gogoproto/proto"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// fixtureDir is the path to the spectre light client test fixtures.
const fixtureDir = "../../spectre/packages/ethereum/light-client/src/test_utils/fixtures"

// stepsFixture mirrors the Rust StepsFixture JSON.
type stepsFixture struct {
	Steps []struct {
		Name string          `json:"name"`
		Data json.RawMessage `json:"data"`
	}
}

// loadFixture loads a fixture file by name.
func loadFixture(t testT, name string) *stepsFixture {
	t.Helper()
	raw, err := os.ReadFile(fmt.Sprintf("%s/%s.json", fixtureDir, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var f stepsFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return &f
}

// testT is the subset of testing.T used by fixture helpers.
type testT interface {
	Fatalf(format string, args ...interface{})
	Helper()
}

// jsonFixedBytes decodes a "0x..." hex string.
func jsonFixedBytes(s string) []byte {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil
	}
	return b
}

// jsonUint decodes a JSON number or decimal string into uint64.
func jsonUint(v interface{}) uint64 {
	switch x := v.(type) {
	case float64:
		return uint64(x)
	case string:
		n, err := strconv.ParseUint(x, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

// rawJSON converts a raw message to a generic map.
func rawJSON(raw json.RawMessage) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// fixtureString reads a string field.
func fixtureString(m map[string]json.RawMessage, key string) string {
	var s string
	if raw, ok := m[key]; ok {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

// fixtureUint reads a uint64 field accepting number-or-string encoding.
func fixtureUint(m map[string]json.RawMessage, key string) uint64 {
	if raw, ok := m[key]; ok {
		var v interface{}
		if err := json.Unmarshal(raw, &v); err == nil {
			return jsonUint(v)
		}
	}
	return 0
}

// fixtureBytes reads a hex string field.
func fixtureBytes(m map[string]json.RawMessage, key string) []byte {
	return jsonFixedBytes(fixtureString(m, key))
}

// fixtureBytesList reads a list of hex strings.
func fixtureBytesList(m map[string]json.RawMessage, key string) [][]byte {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	var strs []string
	if err := json.Unmarshal(raw, &strs); err != nil {
		return nil
	}
	out := make([][]byte, 0, len(strs))
	for _, s := range strs {
		out = append(out, jsonFixedBytes(s))
	}
	return out
}

// decodeForkParameters decodes the fork parameters JSON.
func decodeForkParameters(m map[string]json.RawMessage) *ForkParameters {
	fp := &ForkParameters{
		GenesisForkVersion: fixtureBytes(m, "genesis_fork_version"),
		GenesisSlot:        fixtureUint(m, "genesis_slot"),
		Altair:             *decodeFork(m, "altair"),
		Bellatrix:          *decodeFork(m, "bellatrix"),
		Capella:            *decodeFork(m, "capella"),
		Deneb:              *decodeFork(m, "deneb"),
		Electra:            *decodeFork(m, "electra"),
		Fulu:               *decodeFork(m, "fulu"),
	}
	return fp
}

// decodeFork decodes a single fork entry; a missing entry is unscheduled
// (zero version, epoch = max uint64).
func decodeFork(m map[string]json.RawMessage, key string) *Fork {
	raw, ok := m[key]
	if !ok {
		return unscheduledFork()
	}
	fm := rawJSON(raw)
	fork := &Fork{
		Version: fixtureBytes(fm, "version"),
		Epoch:   fixtureUint(fm, "epoch"),
	}
	if fork.Epoch == ^uint64(0) && len(fork.Version) == 0 {
		return unscheduledFork()
	}
	return fork
}

// unscheduledFork returns a fork that never activates.
func unscheduledFork() *Fork {
	return &Fork{Version: make([]byte, 4), Epoch: ^uint64(0)}
}

// fixtureBool reads a boolean field accepting bool-or-string encoding.
func fixtureBool(m map[string]json.RawMessage, key string) bool {
	if raw, ok := m[key]; ok {
		var v interface{}
		if err := json.Unmarshal(raw, &v); err == nil {
			switch x := v.(type) {
			case bool:
				return x
			case string:
				return strings.EqualFold(x, "true")
			}
		}
	}
	return false
}

// decodeClientState decodes the fixture client state JSON.
func decodeClientState(raw json.RawMessage) *ClientState {
	m := rawJSON(raw)
	return &ClientState{
		ChainId:                      fixtureUint(m, "chain_id"),
		GenesisValidatorsRoot:        fixtureBytes(m, "genesis_validators_root"),
		MinSyncCommitteeParticipants: fixtureUint(m, "min_sync_committee_participants"),
		SyncCommitteeSize:            fixtureUint(m, "sync_committee_size"),
		GenesisTime:                  fixtureUint(m, "genesis_time"),
		GenesisSlot:                  fixtureUint(m, "genesis_slot"),
		ForkParameters:               *decodeForkParameters(rawJSON(m["fork_parameters"])),
		SecondsPerSlot:               fixtureUint(m, "seconds_per_slot"),
		SlotsPerEpoch:                fixtureUint(m, "slots_per_epoch"),
		EpochsPerSyncCommitteePeriod: fixtureUint(m, "epochs_per_sync_committee_period"),
		LatestSlot:                   fixtureUint(m, "latest_slot"),
		LatestExecutionBlockNumber:   fixtureUint(m, "latest_execution_block_number"),
		Frozen:                       fixtureBool(m, "is_frozen"),
		IbcContractAddress:           fixtureBytes(m, "ibc_contract_address"),
		IbcCommitmentSlot:            fixtureBytes(m, "ibc_commitment_slot"),
	}
}

// decodeSummarizedSyncCommittee decodes a committee summary.
func decodeSummarizedSyncCommittee(m map[string]json.RawMessage) *SummarizedSyncCommittee {
	return &SummarizedSyncCommittee{
		PubkeysHash:     fixtureBytes(m, "pubkeys_hash"),
		AggregatePubkey: fixtureBytes(m, "aggregate_pubkey"),
	}
}

// decodeConsensusState decodes the fixture consensus state JSON.
func decodeConsensusState(raw json.RawMessage) *ConsensusState {
	m := rawJSON(raw)
	cs := &ConsensusState{
		Slot:                 fixtureUint(m, "slot"),
		StateRoot:            fixtureBytes(m, "state_root"),
		Timestamp:            fixtureUint(m, "timestamp"),
		CurrentSyncCommittee: *decodeSummarizedSyncCommittee(rawJSON(m["current_sync_committee"])),
	}
	if next, ok := m["next_sync_committee"]; ok && string(next) != "null" {
		cs.NextSyncCommittee = decodeSummarizedSyncCommittee(rawJSON(next))
	}
	return cs
}

// decodeSyncCommittee decodes a full sync committee.
func decodeSyncCommittee(raw json.RawMessage) *SyncCommittee {
	m := rawJSON(raw)
	return &SyncCommittee{
		Pubkeys:         fixtureBytesList(m, "pubkeys"),
		AggregatePubkey: fixtureBytes(m, "aggregate_pubkey"),
	}
}

// decodeLightClientUpdate decodes a light client update.
func decodeLightClientUpdate(raw json.RawMessage) LightClientUpdate {
	m := rawJSON(raw)
	update := LightClientUpdate{
		AttestedHeader:  *decodeLightClientHeader(m["attested_header"]),
		FinalizedHeader: *decodeLightClientHeader(m["finalized_header"]),
		SyncAggregate:   *decodeSyncAggregate(m["sync_aggregate"]),
		SignatureSlot:   fixtureUint(m, "signature_slot"),
	}
	if next, ok := m["next_sync_committee"]; ok && string(next) != "null" {
		update.NextSyncCommittee = decodeSyncCommittee(next)
	}
	if branch, ok := m["next_sync_committee_branch"]; ok && string(branch) != "null" {
		update.NextSyncCommitteeBranch = fixtureBytesList(m, "next_sync_committee_branch")
	}
	update.FinalityBranch = fixtureBytesList(m, "finality_branch")
	return update
}

// decodeSyncAggregate decodes a sync aggregate.
func decodeSyncAggregate(raw json.RawMessage) *SyncAggregate {
	m := rawJSON(raw)
	return &SyncAggregate{
		SyncCommitteeBits:      fixtureBytes(m, "sync_committee_bits"),
		SyncCommitteeSignature: fixtureBytes(m, "sync_committee_signature"),
	}
}

// decodeLightClientHeader decodes a light client header.
func decodeLightClientHeader(raw json.RawMessage) *LightClientHeader {
	m := rawJSON(raw)
	return &LightClientHeader{
		Beacon:          *decodeBeaconBlockHeader(m["beacon"]),
		Execution:       *decodeExecutionPayloadHeader(m["execution"]),
		ExecutionBranch: fixtureBytesList(m, "execution_branch"),
	}
}

// decodeBeaconBlockHeader decodes a beacon block header.
func decodeBeaconBlockHeader(raw json.RawMessage) *BeaconBlockHeader {
	m := rawJSON(raw)
	return &BeaconBlockHeader{
		Slot:          fixtureUint(m, "slot"),
		ProposerIndex: fixtureUint(m, "proposer_index"),
		ParentRoot:    fixtureBytes(m, "parent_root"),
		StateRoot:     fixtureBytes(m, "state_root"),
		BodyRoot:      fixtureBytes(m, "body_root"),
	}
}

// decodeExecutionPayloadHeader decodes an execution payload header. The
// base_fee_per_gas field is a decimal string (U256) and is converted to
// 32-byte big-endian.
func decodeExecutionPayloadHeader(raw json.RawMessage) *ExecutionPayloadHeader {
	m := rawJSON(raw)
	return &ExecutionPayloadHeader{
		ParentHash:       fixtureBytes(m, "parent_hash"),
		FeeRecipient:     fixtureBytes(m, "fee_recipient"),
		StateRoot:        fixtureBytes(m, "state_root"),
		ReceiptsRoot:     fixtureBytes(m, "receipts_root"),
		LogsBloom:        fixtureBytes(m, "logs_bloom"),
		PrevRandao:       fixtureBytes(m, "prev_randao"),
		BlockNumber:      fixtureUint(m, "block_number"),
		GasLimit:         fixtureUint(m, "gas_limit"),
		GasUsed:          fixtureUint(m, "gas_used"),
		Timestamp:        fixtureUint(m, "timestamp"),
		ExtraData:        fixtureBytes(m, "extra_data"),
		BaseFeePerGas:    parseU256To32BE(fixtureString(m, "base_fee_per_gas")),
		BlockHash:        fixtureBytes(m, "block_hash"),
		TransactionsRoot: fixtureBytes(m, "transactions_root"),
		WithdrawalsRoot:  fixtureBytes(m, "withdrawals_root"),
		BlobGasUsed:      fixtureUint(m, "blob_gas_used"),
		ExcessBlobGas:    fixtureUint(m, "excess_blob_gas"),
	}
}

// parseU256To32BE converts a decimal or 0x-hex string to a 32-byte
// big-endian value.
func parseU256To32BE(s string) []byte {
	s = strings.TrimSpace(s)
	base := 10
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		base = 16
		s = s[2:]
	}
	n, ok := new(big.Int).SetString(s, base)
	if !ok {
		return make([]byte, 32)
	}
	out := make([]byte, 32)
	n.FillBytes(out)
	return out
}

// fixtureHeader mirrors the JSON Header client message.
type fixtureHeader struct {
	TrustedSlot         uint64
	ActiveSyncCommittee *SyncCommittee
	IsNextCommittee     bool
	ConsensusUpdate     LightClientUpdate
}

// decodeHeader decodes a Header client message from its JSON encoding. The
// active sync committee is an externally tagged enum: {"Next": {...}} or
// {"Current": {...}}.
func decodeHeader(raw json.RawMessage) *fixtureHeader {
	m := rawJSON(raw)
	h := &fixtureHeader{
		TrustedSlot:     fixtureUint(m, "trusted_slot"),
		ConsensusUpdate: decodeLightClientUpdate(m["consensus_update"]),
	}
	if asc, ok := m["active_sync_committee"]; ok {
		tagged := rawJSON(asc)
		if committee, isNext := tagged["Next"]; isNext {
			h.ActiveSyncCommittee = decodeSyncCommittee(committee)
			h.IsNextCommittee = true
		} else if committee, isCurrent := tagged["Current"]; isCurrent {
			h.ActiveSyncCommittee = decodeSyncCommittee(committee)
		}
	}
	return h
}

// toProto converts the fixture header to the protobuf Header.
func (h *fixtureHeader) toProto() *Header {
	return &Header{
		TrustedSlot:         h.TrustedSlot,
		ActiveSyncCommittee: h.ActiveSyncCommittee,
		IsNextCommittee:     h.IsNextCommittee,
		ConsensusUpdate:     h.ConsensusUpdate,
	}
}

// fixtureHeaderRaw extracts the raw JSON of the first header in a fixture
// step's relayer tx body.
func fixtureHeaderRaw(t testT, fixture *stepsFixture, step int) json.RawMessage {
	t.Helper()
	m := rawJSON(fixture.Steps[step].Data)
	txHex := fixtureString(m, "relayer_tx_body")
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		t.Fatalf("decode relayer tx body: %v", err)
		return nil
	}
	var body tx.TxBody
	if err := proto.Unmarshal(txBytes, &body); err != nil {
		t.Fatalf("unmarshal tx body: %v", err)
		return nil
	}
	for _, any := range body.Messages {
		if any.TypeUrl != "/ibc.core.client.v1.MsgUpdateClient" {
			continue
		}
		var uc clienttypes.MsgUpdateClient
		if err := proto.Unmarshal(any.Value, &uc); err != nil {
			t.Fatalf("unmarshal MsgUpdateClient: %v", err)
			return nil
		}
		data, err := wasmClientMessageData(uc.ClientMessage.Value)
		if err != nil {
			t.Fatalf("extract wasm client message data: %v", err)
			return nil
		}
		return data
	}
	t.Fatalf("no update client message in step %d", step)
	return nil
}

// membershipProofJSON is the JSON membership proof (eth_getProof shape).
type membershipProofJSON struct {
	AccountProof struct {
		StorageRoot string   `json:"storage_root"`
		Proof       []string `json:"proof"`
	} `json:"account_proof"`
	StorageProof struct {
		Key   string   `json:"key"`
		Value string   `json:"value"`
		Proof []string `json:"proof"`
	} `json:"storage_proof"`
}

// decodeMembershipProof converts the JSON membership proof into the protobuf
// MembershipProof.
func decodeMembershipProof(raw []byte) *MembershipProof {
	var mpj membershipProofJSON
	if err := json.Unmarshal(raw, &mpj); err != nil {
		return nil
	}
	mp := &MembershipProof{
		AccountProof: AccountProof{
			StorageRoot: jsonFixedBytes(mpj.AccountProof.StorageRoot),
		},
		StorageProof: StorageProof{
			Key:   jsonFixedBytes(mpj.StorageProof.Key),
			Value: parseU256To32BE(mpj.StorageProof.Value),
		},
	}
	for _, n := range mpj.AccountProof.Proof {
		mp.AccountProof.Proof = append(mp.AccountProof.Proof, jsonFixedBytes(n))
	}
	for _, n := range mpj.StorageProof.Proof {
		mp.StorageProof.Proof = append(mp.StorageProof.Proof, jsonFixedBytes(n))
	}
	return mp
}
