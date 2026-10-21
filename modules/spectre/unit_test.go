package spectre

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
)

// TestSyncCommitteeRootVector checks the sync committee hash tree root
// against the known test vector in the spectre ethereum types unit tests.
func TestSyncCommitteeRootVector(t *testing.T) {
	pubkeysHex := []string{
		"81ea9f74ef7d935b807474e38954ae3934856219a23e074954b2e860c5a3c400f9aedb42cd27cb4ceb697ca36d1e58cb",
		"84d08d58c31bcd3cddf93e13d6f50203897384afa34644bff1135efe8e01c81c6a91ca6c234bb1e51ca32e41b828aaf9",
		"a759f6bcca8f35fcaadc406cc4b828c016c0ed23882987a79f52f2933b5cedefe24e31df6fd0d38e8a802dbafd750d01",
		"8d028a021c5c31a1aa1e18eda74cfaf0fba1c454c17c2e0fc730dd07a19d0c77f7a905d54017292f3e800ca06b6977cd",
		"b27ad13afc8ff30e087797b344c8382bb0a84447549f1b0274059ddd652276e7b148ba8808a10cc45746762957d4efbe",
		"a804e4fa8d1391a9d078aa93985a12503b84ce4f6f1f9e70ab7fca421e1cf972538666299d4c1bfc39327b469b2db7a8",
		"996323af7e545fb6363ace53f1538c7ddc3eb0d985b2479da3ee4ace10cbc393b518bf02d1a2ddb2f5bdf09b473933ea",
		"96947de9e6068c22a7716656a2755a9551b0b66c2d1a741bf84a088fe1e840e992dc39861bf8ba3e8d5b6d21e8f57e64",
		"ae5302796cfeca685eaf37ffd5baeb32121f2f07415bee26cc0051ee513ff3932d2c365e3d9f87b0949a5980445cb64c",
		"996d10c3026b9344532b06c70a596f972a1e779a1f6106d3da9f6ba376bbf7ec82d2f52629e5dbf3f7d03b00f6b862af",
		"a35c6004f387430c3797ab0157af7b824c8fe106241c7cdeb897d900c0f9e4bb945ff2a6b88cbd10e35ec48aaa554ecb",
		"abd12678c73463ecea5867a80caf256d5c5e6ba53ff188b143a4d5be83365ad257edf39eaa1ba8753c4cdf4c632ff99e",
		"81fa222737fe818b43f55f209f42adaee135b2801d02709617fc88c2871852358260ace97cf323e761b5cc18bc7325b3",
		"ab64f900c770e2b99de6b86b4390bbd1579bd48dccec55800adbcf52e006f22128e9971bbf3a92cc0105b0974849935a",
		"930743bfc7e18d3bd7351eaa74f477505268c1e4e1fd1ca3ccccdefb2595517343bbb8f5589c435c3c39323a4c0080f8",
		"ab72cbc6575c3179680a58c0ecd5de46d2678ccbafc016746348ee5688edcb21b4e15bd37c70c508e3ea73103c2d566b",
		"84dc37ca3cd621d3da0fbdd11ca84021e0cd81a73d772dd6fcf19775b72eb64af4e573213378ccee0915dde92ac83ba6",
		"8d46e9aa0c1986056e407efc7013b7f271027d3c98ce96667faa98074ab0588a61681faf78644c11819a459a95689dab",
		"b5e898a1fc06d51c695712928f44646d15451340d1b3e480a40f03250160bc07d3b6691ec94361dd524d59d9df7f76d3",
		"a4ee6d37dc259cbb5237e4265429a9fd8ab5643af81628cc101e0d8b4a333ef2618a37df89ea3f92b5ea4333d8cda393",
		"8aa5bbee21e98c7b9e7a4c8ea45aa99f89e22992fa4fc2d73869d77da4cc8a05b25b61931ff521986677dd7f7159e8e6",
		"91709ee06497b9ac049325853d64947290189a8c2322e3a500d91e23ea02dc158b6db63ae558b3b7670357a151cd6071",
		"8fda66b8607af873f4c2c8218dd3ffc7940d411047eb199b5cd010156af4845d21dd2e65b0e44cfffb5e78271e9bb29d",
		"b72cb106b7bc1ecae219e0ae1830a509ed18a042b56a2779f4033419de69ba8ae8017090caed1f5377bfa68506157360",
		"896a51e0b0de0f29029af38b796db1f1e6d0f9f9085ade40a313a60cb723fa3d58f6587175570086c4fbf0fe5331f1c8",
		"aaf6c1251e73fb600624937760fef218aace5b253bf068ed45398aeb29d821e4d2899343ddcbbe37cb3f6cf500dff26c",
		"9918433b8f0bc5e126da3fdef8d7b71456492dae6d2d07f2e10c7a7f852046f84ed0ce6d3bfec42200670db27dcf3037",
		"a03c2a82374e04b2e0594c4ce14fb3f225b46f13188f0d8002a523c7dcfb939ae4856053c2c9c695374d7c3685df1ca5",
		"8d8985e5dd341c9035b37bf7391c5944c28131b47c7d5359d18fca598010ba9a63e27c55e6b421a807038c320564db17",
		"b24391aa97bfff29adc935d06a2b6d583433caf82f92de1980e0192d3b270323bdbf24b86dc61520a40c419dde3df4b3",
		"af61f263addfb41c46d66e60ecfb598a5942f648f58718b6b4e4c92019fdb12328efbff98703134bcf28e9c1fab4bb60",
		"b63f327df68581cdc02a66c1c65e906a06a1a3a8d7a6e38f7b6da944e8e6cc2db85fced5327d8c12945ceb33018272ca",
	}
	pubkeys := make([][]byte, len(pubkeysHex))
	for i, p := range pubkeysHex {
		b, err := hex.DecodeString(p)
		require.NoError(t, err)
		pubkeys[i] = b
	}
	agg, err := hex.DecodeString("a7b9141877f397e9d2a36cd86407387bbcec6d557b30ccd9e62adca217e458d7495b581e048fa1084218cadf8f45b9ff")
	require.NoError(t, err)

	sc := &SyncCommittee{Pubkeys: pubkeys, AggregatePubkey: agg}
	root := syncCommitteeRoot(sc)

	expected, err := hex.DecodeString("5361eb179f7499edbf09e514d317002f1d365d72e14a56c931e9edaccca3ff29")
	require.NoError(t, err)
	require.Equal(t, expected, root[:])
}

// TestComputeDomainVector checks the sync-committee domain derivation
// against the known test vector in the spectre ethereum types unit tests.
func TestComputeDomainVector(t *testing.T) {
	cs := &ClientState{
		GenesisValidatorsRoot: mustHex(t, "d61ea484febacfae5298d52a2b581f3e305a51f3112a9241b968dccf019f7b11"),
		ForkParameters: ForkParameters{
			GenesisForkVersion: mustHex(t, "00000001"),
			Electra:            Fork{Version: mustHex(t, "04000001")},
		},
	}
	domain := cs.computeDomain(domainTypeSyncCommittee, cs.ForkParameters.Electra.Version)
	expected := mustHex(t, "07000000eaa5664b85c5e9dc16d64ac6ee15cc92ec477990061b30024696db67")
	require.Equal(t, expected, domain[:])
}

// TestValidateMerkleBranch checks branch walking, including the depth-2
// normalization vector from the spectre tree hash unit tests.
func TestValidateMerkleBranch(t *testing.T) {
	branch := [][]byte{mustHex(t, "75d7411cb01daad167713b5a9b7219670f0e500653cbbcd45cfe1bfe04222459")}

	normalized, err := normalizeMerkleBranch(bytesTo32Array(branch), 4)
	require.NoError(t, err)
	require.Len(t, normalized, 2)
	require.Equal(t, [32]byte{}, normalized[0])
	require.Equal(t, bytes32(branch[0]), normalized[1])

	// A branch longer than the gindex depth must be rejected.
	_, err = normalizeMerkleBranch(make([][32]byte, 3), 2)
	require.Error(t, err)

	// Shorter branches are rejected by validation.
	err = validateMerkleBranch([32]byte{}, bytesTo32Array(branch), 8, [32]byte{})
	require.Error(t, err)
}

// TestComputeForkVersion checks the fork version selection order, including
// the unscheduled-fulu sentinel.
func TestComputeForkVersion(t *testing.T) {
	v := func(n byte) []byte { return []byte{n, 0, 0, 0} }
	fp := &ForkParameters{
		GenesisForkVersion: v(0),
		Altair:             Fork{Version: v(1)},
		Bellatrix:          Fork{Version: v(2)},
		Capella:            Fork{Version: v(3)},
		Deneb:              Fork{Version: v(4)},
		Electra:            Fork{Version: v(5), Epoch: 100},
		Fulu:               Fork{Version: v(6), Epoch: ^uint64(0)},
	}
	require.Equal(t, v(5), fp.computeForkVersion(150))
	require.Equal(t, v(5), fp.computeForkVersion(1_000_000), "unscheduled fulu must never be selected")
	require.Equal(t, v(4), fp.computeForkVersion(100-1))
	require.Equal(t, v(4), fp.computeForkVersion(0), "deneb at epoch 0 covers the genesis range")
}

// TestFixtureNegativeChecks verifies that tampered updates and proofs are
// rejected.
func TestFixtureNegativeChecks(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	runner := newFixtureRunner(t, fixture)
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)

	header := msgs.headers[0]
	trusted := runner.consStates[header.TrustedSlot]
	currentTime := header.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000

	// A valid header verifies.
	require.NoError(t, runner.clientState.verifyHeader(trusted, currentTime, header.toProto()))

	// Corrupting the finalized state root must fail verification.
	tampered := header.toProto()
	tampered.ConsensusUpdate.FinalizedHeader.Execution.StateRoot[0] ^= 0xff
	require.Error(t, runner.clientState.verifyHeader(trusted, currentTime, tampered))

	// Corrupting the sync signature must fail verification.
	tampered = header.toProto()
	tampered.ConsensusUpdate.SyncAggregate.SyncCommitteeSignature[0] ^= 0xff
	require.Error(t, runner.clientState.verifyHeader(trusted, currentTime, tampered))

	// An update must advance the trusted slot.
	stale := header.toProto()
	stale.ConsensusUpdate.FinalizedHeader.Beacon.Slot = trusted.Slot
	require.Error(t, runner.clientState.verifyHeader(trusted, currentTime, stale))

	// Frozen clients reject updates.
	frozen := runner.clientState.Clone()
	frozen.Frozen = true
	require.Error(t, frozen.verifyHeader(trusted, currentTime, header.toProto()))
}

// TestFixtureMembershipNegative verifies tampered membership proofs fail.
func TestFixtureMembershipNegative(t *testing.T) {
	fixture := loadFixture(t, "Test_ICS20TransferERC20TokenfromEthereumToCosmosAndBack")
	runner := newFixtureRunner(t, fixture)

	var rp *channeltypesv2.MsgRecvPacket
	for i := 1; i < len(fixture.Steps); i++ {
		msgs := decodeRelayerMessages(t, fixture.Steps[i].Data)
		if len(msgs.headers) > 0 {
			runner.applyUpdates(msgs)
		}
		for _, m := range msgs.recv {
			if rp == nil {
				rp = m
			}
		}
	}
	require.NotNil(t, rp, "fixture contained no recv packets")

	proof := decodeMembershipProof(rp.ProofCommitment)
	consState := runner.consStates[rp.ProofHeight.GetRevisionHeight()]
	path := [][]byte{hostv2.PacketCommitmentKey(rp.Packet.SourceClient, rp.Packet.Sequence)}
	value := channeltypesv2.CommitPacket(rp.Packet)

	require.NoError(t, verifyStorageMembership(runner.clientState, consState, proof, path, value))

	// A wrong expected value must fail.
	badValue := append([]byte(nil), value...)
	badValue[0] ^= 0xff
	require.Error(t, verifyStorageMembership(runner.clientState, consState, proof, path, badValue))

	// A wrong path must fail.
	badPath := [][]byte{append([]byte(nil), path[0]...)}
	badPath[0][0] ^= 0xff
	require.Error(t, verifyStorageMembership(runner.clientState, consState, proof, badPath, value))

	// A tampered proof node must fail.
	tampered := decodeMembershipProof(rp.ProofCommitment)
	tampered.AccountProof.Proof[0][0] ^= 0xff
	require.Error(t, verifyStorageMembership(runner.clientState, consState, tampered, path, value))

	// Tampered storage proof nodes must fail.
	tampered = decodeMembershipProof(rp.ProofCommitment)
	tampered.StorageProof.Proof[len(tampered.StorageProof.Proof)-1][0] ^= 0xff
	require.Error(t, verifyStorageMembership(runner.clientState, consState, tampered, path, value))
}

// mustHex decodes a hex string.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}
