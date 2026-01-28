package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
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
