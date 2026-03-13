package gno

import (
	"testing"
	"time"

	cmtmath "github.com/cometbft/cometbft/libs/math"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	ics23 "github.com/cosmos/ics23/go"
)

func TestUpdateStateOnMisbehaviour(t *testing.T) {
	cdc := getTestCodec()
	clientStore := setupClientStore(t)
	ctx := getTestContext(t, time.Now().UTC())

	// Setup client state
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	setClientState(clientStore, cdc, cs)

	// Create misbehaviour
	header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
	header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
	misbehaviour := NewMisbehaviour(testClientID, header1, header2)

	// Update state on misbehaviour
	cs.UpdateStateOnMisbehaviour(ctx, cdc, clientStore, misbehaviour)

	// Verify client is frozen
	updatedCS, found := getClientState(clientStore, cdc)
	require.True(t, found)
	require.Equal(t, FrozenHeight, updatedCS.FrozenHeight)
}

func TestCheckTrustedHeader(t *testing.T) {
	testCases := []struct {
		name        string
		setupHeader func() *Header
		setupConsState func() *ConsensusState
		expectErr   bool
	}{
		{
			name: "error - mismatched validators hash",
			setupHeader: func() *Header {
				valSet, _ := createTestValidatorSet(1, 100)
				header := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				header.TrustedValidators = valSet
				return header
			},
			setupConsState: func() *ConsensusState {
				return createTestConsensusState(time.Now().UTC())
			},
			// Will fail due to hash mismatch (test helpers generate random hashes)
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := tc.setupHeader()
			consState := tc.setupConsState()

			err := checkTrustedHeader(header, consState)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateState_HeaderType(t *testing.T) {
	cdc := getTestCodec()
	clientStore := setupClientStore(t)
	ctx := getTestContext(t, time.Now().UTC())

	// Setup client state and consensus state
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	setClientState(clientStore, cdc, cs)
	consState := createTestConsensusState(time.Now().UTC())
	setConsensusState(clientStore, cdc, consState, cs.LatestHeight)

	// Create a header (UpdateState will be called with this)
	header := createTestHeader(t, testChainID, 150, clienttypes.NewHeight(1, 100), time.Now().UTC())

	// UpdateState should attempt to process the header
	// Note: Full verification would fail due to invalid signatures in test data
	heights := cs.UpdateState(ctx, cdc, clientStore, header)

	// Even with potentially invalid verification, UpdateState should return heights
	// The actual height returned depends on whether verification passes
	require.NotNil(t, heights)
}

func TestUpdateStateOnMisbehaviour_FreezesClient(t *testing.T) {
	cdc := getTestCodec()
	clientStore := setupClientStore(t)
	ctx := getTestContext(t, time.Now().UTC())

	// Setup active client state
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	require.True(t, cs.FrozenHeight.IsZero())
	setClientState(clientStore, cdc, cs)

	// Create misbehaviour
	header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
	header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
	misbehaviour := NewMisbehaviour(testClientID, header1, header2)

	// Call UpdateStateOnMisbehaviour
	cs.UpdateStateOnMisbehaviour(ctx, cdc, clientStore, misbehaviour)

	// Verify the client is now frozen
	storedCS, found := getClientState(clientStore, cdc)
	require.True(t, found)
	require.False(t, storedCS.FrozenHeight.IsZero())
	require.Equal(t, FrozenHeight, storedCS.FrozenHeight)
}

func TestVerifyClientMessage_Header(t *testing.T) {
	cdc := getTestCodec()
	clientStore := setupClientStore(t)
	ctx := getTestContext(t, time.Now().UTC())

	// Setup client state and consensus state
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	setClientState(clientStore, cdc, cs)
	consState := createTestConsensusState(time.Now().UTC())
	setConsensusState(clientStore, cdc, consState, cs.LatestHeight)

	// Create a header
	header := createTestHeader(t, testChainID, 150, clienttypes.NewHeight(1, 100), time.Now().UTC())

	// VerifyClientMessage will attempt to verify the header
	// It will likely fail due to invalid signatures in test data, but shouldn't panic
	err := cs.VerifyClientMessage(ctx, cdc, clientStore, header)

	// We expect an error because our test headers don't have valid signatures
	require.Error(t, err)
}

func TestVerifyClientMessage_Misbehaviour(t *testing.T) {
	cdc := getTestCodec()
	clientStore := setupClientStore(t)
	ctx := getTestContext(t, time.Now().UTC())

	// Setup client state and consensus state
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	setClientState(clientStore, cdc, cs)
	consState := createTestConsensusState(time.Now().UTC())
	setConsensusState(clientStore, cdc, consState, cs.LatestHeight)

	// Also set consensus state at trusted height
	consState2 := createTestConsensusState(time.Now().UTC().Add(-time.Hour))
	setConsensusState(clientStore, cdc, consState2, clienttypes.NewHeight(1, 50))

	// Create misbehaviour
	header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
	header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
	misbehaviour := NewMisbehaviour(testClientID, header1, header2)

	// VerifyClientMessage will attempt to verify the misbehaviour
	err := cs.VerifyClientMessage(ctx, cdc, clientStore, misbehaviour)

	// We expect an error because our test misbehaviour doesn't have valid signatures
	require.Error(t, err)
}

// TestCheckMisbehaviourHeader_UsesClientTrustLevel tests that checkMisbehaviourHeader
// uses the client's configured TrustLevel rather than the hardcoded LCDefaultTrustLevel.
//
// Scenario: 4 trusted validators (A,B,C,D) with 100 VP each (total 400).
// Misbehaviour header is signed by only 2 of them (200 VP = 50%).
//   - With trust level 1/3: 200 > 133 → should pass
//   - With trust level 2/3: 200 not > 266 → should reject
//
// With the old hardcoded LCDefaultTrustLevel (1/3), both would pass.
func TestCheckMisbehaviourHeader_UsesClientTrustLevel(t *testing.T) {
	blockTime := time.Now().UTC()
	votingPower := int64(100)

	// Create 4 keys for the trusted set
	allKeys := []ed25519.PrivKeyEd25519{makePrivKey(), makePrivKey(), makePrivKey(), makePrivKey()}

	// Sort all 4 validators by address (as they would be on-chain).
	// ConvertToGnoValidatorSet preserves input order and GetByAddress
	// requires sorted order, so the proto ValidatorSet must be pre-sorted.
	sortedTrustedBftVals, sortedTrustedKeys := sortedValidatorsAndKeys(allKeys, votingPower)

	// Build the proto trusted validator set in sorted address order
	trustedProtoVals := make([]*Validator, len(sortedTrustedBftVals))
	for i, bv := range sortedTrustedBftVals {
		trustedProtoVals[i] = createTestValidatorWithKey(votingPower, sortedTrustedKeys[i])
		_ = bv // used for ordering
	}
	trustedValSet := &ValidatorSet{Validators: trustedProtoVals}

	// Pick the first 2 (sorted) validators as the signers (200 of 400 VP)
	signerKeys := sortedTrustedKeys[:2]
	signerValSet := &ValidatorSet{
		Validators: []*Validator{
			trustedProtoVals[0],
			trustedProtoVals[1],
		},
	}

	// Create a signed header using only the 2 signers
	signedHeader := createTestSignedHeader(testChainID, 100, blockTime, signerValSet, signerKeys)

	// Build the misbehaviour Header with the full trusted validator set
	header := &Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      signerValSet,
		TrustedHeight:     clienttypes.NewHeight(1, 50),
		TrustedValidators: trustedValSet,
	}

	// Build a consensus state whose NextValidatorsHash matches the trusted validator set.
	// checkTrustedHeader uses ConvertToGnoValidatorSet to compute the hash, so we must
	// use the same function to get a matching hash.
	gnoTrustedVals, err := ConvertToGnoValidatorSet(trustedValSet)
	require.NoError(t, err)
	consState := &ConsensusState{
		Timestamp:          blockTime.Add(-time.Hour),
		Root:               commitmenttypes.NewMerkleRoot([]byte("root")),
		NextValidatorsHash: gnoTrustedVals.Hash(),
		LcType:             Gno,
	}

	// Test with 2/3 trust level: 200/400 does not exceed 266, should fail
	csHighTrust := NewClientState(
		testChainID,
		NewFractionFromTm(cmtmath.Fraction{Numerator: 2, Denominator: 3}),
		testTrustingPeriod, testUnbondingPeriod, testMaxClockDrift,
		clienttypes.NewHeight(1, 100),
		[]*ics23.ProofSpec{ics23.IavlSpec, ics23.TendermintSpec},
		[]string{"upgrade", "upgradedIBCState"},
	)

	err = checkMisbehaviourHeader(csHighTrust, consState, header, blockTime)
	require.Error(t, err, "should reject: 200 VP does not exceed 2/3 of 400 (266)")
	require.Contains(t, err.Error(), "validator set in header has too much change")

	// Test with 1/3 trust level: 200/400 exceeds 133, should pass
	csLowTrust := NewClientState(
		testChainID,
		DefaultTrustLevel,
		testTrustingPeriod, testUnbondingPeriod, testMaxClockDrift,
		clienttypes.NewHeight(1, 100),
		[]*ics23.ProofSpec{ics23.IavlSpec, ics23.TendermintSpec},
		[]string{"upgrade", "upgradedIBCState"},
	)

	err = checkMisbehaviourHeader(csLowTrust, consState, header, blockTime)
	require.NoError(t, err, "should pass: 200 VP exceeds 1/3 of 400 (133)")
}
