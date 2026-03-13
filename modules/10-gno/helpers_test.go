package gno

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestConvertToGnoCommit_AbsentValidators tests that ConvertToGnoCommit correctly
// handles absent validators represented as zero-value CommitSig structs.
// Proto3 repeated message fields always deserialize as non-nil pointers, so absent
// validators appear as zero-value CommitSig{} rather than nil entries. Without the
// fix, this would panic due to nil BlockId dereference and MustAddressFromString("").
func TestConvertToGnoCommit_AbsentValidators(t *testing.T) {
	chainID := testChainID
	height := int64(10)
	blockTime := time.Now().UTC()

	// Create a validator set with 3 validators, but only 2 will sign
	valSet, privKeys := createTestValidatorSet(3, 100)
	signedHeader := createTestSignedHeader(chainID, height, blockTime, valSet, privKeys)

	// Simulate what a proto3 deserialization produces for absent validators:
	// replace the 3rd precommit with a zero-value CommitSig (not nil).
	signedHeader.Commit.Precommits[2] = &CommitSig{}

	// This must not panic
	gnoCommit, err := ConvertToGnoCommit(signedHeader.Commit)
	require.NoError(t, err)
	require.NotNil(t, gnoCommit)

	// The absent validator's slot should be nil in the converted commit
	require.Nil(t, gnoCommit.Precommits[2], "absent validator should produce nil precommit entry")

	// The present validators should have been converted correctly
	require.NotNil(t, gnoCommit.Precommits[0])
	require.NotNil(t, gnoCommit.Precommits[1])
}

// TestConvertToGnoCommit_AllAbsent tests that a commit with all zero-value
// CommitSig entries (all absent) does not panic.
func TestConvertToGnoCommit_AllAbsent(t *testing.T) {
	commit := &Commit{
		BlockId: createTestBlockID(),
		Precommits: []*CommitSig{
			{}, // zero-value: absent validator
			{}, // zero-value: absent validator
			{}, // zero-value: absent validator
		},
	}

	gnoCommit, err := ConvertToGnoCommit(commit)
	require.NoError(t, err)
	require.NotNil(t, gnoCommit)

	for i, pc := range gnoCommit.Precommits {
		require.Nil(t, pc, "precommit %d should be nil for absent validator", i)
	}
}

// TestConvertToGnoHeader_AppVersion tests that ConvertToGnoHeader preserves
// the AppVersion field. AppVersion is included in the header's Merkle hash
// (the 7th leaf in Header.Hash). If it were dropped, the converted header's
// hash would not match the original commit's BlockID.Hash, and
// SignedHeader.ValidateBasic would reject every header.
func TestConvertToGnoHeader_AppVersion(t *testing.T) {
	appHash := make([]byte, 32)
	rand.Read(appHash)
	consensusHash := make([]byte, 32)
	rand.Read(consensusHash)
	valsHash := make([]byte, 32)
	rand.Read(valsHash)

	protoHeader := &GnoHeader{
		Version:    "1.0.0",
		ChainId:    testChainID,
		Height:     100,
		Time:       time.Now().UTC(),
		AppVersion: "v1.2.3",
		LastBlockId: &BlockID{
			Hash:        make([]byte, 32),
			PartsHeader: &PartSetHeader{Total: 1, Hash: make([]byte, 32)},
		},
		ValidatorsHash:     valsHash,
		NextValidatorsHash: valsHash,
		ConsensusHash:      consensusHash,
		AppHash:            appHash,
		ProposerAddress:    "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5",
	}

	bftHeader, err := ConvertToGnoHeader(protoHeader)
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", bftHeader.AppVersion,
		"AppVersion must be preserved; it is the 7th Merkle leaf in Header.Hash")

	// Verify the hash changes when AppVersion differs
	hashWith := bftHeader.Hash()

	bftHeader.AppVersion = ""
	hashWithout := bftHeader.Hash()

	require.NotEqual(t, hashWith, hashWithout,
		"header hash must differ when AppVersion changes, proving it participates in the Merkle tree")
}
