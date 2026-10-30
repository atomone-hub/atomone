package spectre

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// goldenFixtures are the vendored fixtures whose protobuf encodings the spectre
// relayer's conversion tests compare against.
var goldenFixtures = []string{
	"Test_ICS20TransferERC20TokenfromEthereumToCosmosAndBack",
	"Test_TimeoutPacketFromCosmos",
}

// TestWriteRelayerGolden writes the protobuf encodings of the fixture client
// state, consensus state, headers and storage proofs, as this module's own
// decoders produce them from the Rust client's JSON, into the directory named
// by SPECTRE_WRITE_GOLDEN. The spectre relayer converts the same JSON with an
// independent implementation and must produce the same bytes
// (relayer/client/spectre_proto_test.go): the files are the contract between
// the two repositories. Skipped unless the variable is set.
//
//	SPECTRE_WRITE_GOLDEN=<spectre>/relayer/client/testdata/spectre go test ./modules/spectre -run TestWriteRelayerGolden
func TestWriteRelayerGolden(t *testing.T) {
	root := os.Getenv("SPECTRE_WRITE_GOLDEN")
	if root == "" {
		t.Skip("set SPECTRE_WRITE_GOLDEN=<dir> to write the golden files")
	}

	for _, name := range goldenFixtures {
		fixture := loadFixture(t, name)
		dir := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		write := func(file string, msg interface{ Marshal() ([]byte, error) }) {
			bz, err := msg.Marshal()
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, file), bz, 0o644))
		}

		initial := rawJSON(fixture.Steps[0].Data)
		write("client_state.pb", decodeClientState(initial["client_state"]))
		write("consensus_state.pb", decodeConsensusState(initial["consensus_state"]))

		var headers, recv, acks, timeouts int
		for i := 1; i < len(fixture.Steps); i++ {
			msgs := decodeRelayerMessages(t, fixture.Steps[i].Data)
			for _, h := range msgs.headers {
				write(fmt.Sprintf("header_%d.pb", headers), h.toProto())
				headers++
			}
			for _, m := range msgs.recv {
				write(fmt.Sprintf("recv_proof_%d.pb", recv), decodeMembershipProof(m.ProofCommitment))
				recv++
			}
			for _, m := range msgs.ack {
				write(fmt.Sprintf("ack_proof_%d.pb", acks), decodeMembershipProof(m.ProofAcked))
				acks++
			}
			for _, m := range msgs.timeout {
				write(fmt.Sprintf("timeout_proof_%d.pb", timeouts), decodeMembershipProof(m.ProofUnreceived))
				timeouts++
			}
		}
		t.Logf("%s: %d headers, %d recv, %d ack, %d timeout proofs", name, headers, recv, acks, timeouts)
	}
}
