package spectre

import (
	"testing"
)

// ExportLoadFixtureBootstrap returns the marshaled bootstrap of a fixture and
// its update headers, so external tests (package spectre_test) can drive the
// app-level IBC plumbing with real fixture data.
func ExportLoadFixtureBootstrap(t *testing.T, fixtureName string) (clientStateBz, consensusStateBz []byte, headers []*Header) {
	t.Helper()
	fixture := loadFixture(t, fixtureName)
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	cons := decodeConsensusState(initial["consensus_state"])

	var err error
	clientStateBz, err = cs.Marshal()
	if err != nil {
		t.Fatalf("marshal client state: %v", err)
	}
	consensusStateBz, err = cons.Marshal()
	if err != nil {
		t.Fatalf("marshal consensus state: %v", err)
	}
	for i := 1; i < len(fixture.Steps); i++ {
		for _, fh := range decodeRelayerMessages(t, fixture.Steps[i].Data).headers {
			headers = append(headers, fh.toProto())
		}
	}
	return clientStateBz, consensusStateBz, headers
}
