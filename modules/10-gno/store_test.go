package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func TestSetAndGetClientState(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()

	// Initially no client state
	_, found := getClientState(clientStore, cdc)
	require.False(t, found)

	// Set client state
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	setClientState(clientStore, cdc, cs)

	// Get client state
	storedCS, found := getClientState(clientStore, cdc)
	require.True(t, found)
	require.Equal(t, cs.ChainId, storedCS.ChainId)
	require.Equal(t, cs.TrustLevel, storedCS.TrustLevel)
	require.Equal(t, cs.LatestHeight, storedCS.LatestHeight)
}

func TestSetAndGetConsensusState(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()
	height := clienttypes.NewHeight(1, 100)

	// Initially no consensus state
	_, found := GetConsensusState(clientStore, cdc, height)
	require.False(t, found)

	// Set consensus state
	cs := createTestConsensusState(time.Now().UTC())
	setConsensusState(clientStore, cdc, cs, height)

	// Get consensus state
	storedCS, found := GetConsensusState(clientStore, cdc, height)
	require.True(t, found)
	require.Equal(t, cs.Timestamp, storedCS.Timestamp)
	require.Equal(t, cs.NextValidatorsHash, storedCS.NextValidatorsHash)
}

func TestDeleteConsensusState(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()
	height := clienttypes.NewHeight(1, 100)

	// Set consensus state
	cs := createTestConsensusState(time.Now().UTC())
	setConsensusState(clientStore, cdc, cs, height)

	// Verify it's there
	_, found := GetConsensusState(clientStore, cdc, height)
	require.True(t, found)

	// Delete it
	deleteConsensusState(clientStore, height)

	// Verify it's gone
	_, found = GetConsensusState(clientStore, cdc, height)
	require.False(t, found)
}

func TestSetAndGetProcessedTime(t *testing.T) {
	clientStore := setupClientStore(t)
	height := clienttypes.NewHeight(1, 100)

	// Initially no processed time
	_, found := GetProcessedTime(clientStore, height)
	require.False(t, found)

	// Set processed time
	processedTime := uint64(time.Now().UnixNano())
	SetProcessedTime(clientStore, height, processedTime)

	// Get processed time
	storedTime, found := GetProcessedTime(clientStore, height)
	require.True(t, found)
	require.Equal(t, processedTime, storedTime)
}

func TestSetAndGetProcessedHeight(t *testing.T) {
	clientStore := setupClientStore(t)
	consHeight := clienttypes.NewHeight(1, 100)
	processedHeight := clienttypes.NewHeight(0, 50)

	// Initially no processed height
	_, found := GetProcessedHeight(clientStore, consHeight)
	require.False(t, found)

	// Set processed height
	SetProcessedHeight(clientStore, consHeight, processedHeight)

	// Get processed height
	storedHeight, found := GetProcessedHeight(clientStore, consHeight)
	require.True(t, found)
	require.Equal(t, processedHeight.GetRevisionNumber(), storedHeight.GetRevisionNumber())
	require.Equal(t, processedHeight.GetRevisionHeight(), storedHeight.GetRevisionHeight())
}

func TestIterationKey(t *testing.T) {
	height1 := clienttypes.NewHeight(0, 100)
	height2 := clienttypes.NewHeight(0, 200)
	height3 := clienttypes.NewHeight(1, 50)

	key1 := IterationKey(height1)
	key2 := IterationKey(height2)
	key3 := IterationKey(height3)

	// Keys should be different
	require.NotEqual(t, key1, key2)
	require.NotEqual(t, key1, key3)
	require.NotEqual(t, key2, key3)

	// Keys should start with prefix
	require.Contains(t, string(key1), KeyIterateConsensusStatePrefix)
}

func TestGetHeightFromIterationKey(t *testing.T) {
	testCases := []struct {
		name   string
		height clienttypes.Height
	}{
		{
			name:   "revision 0, height 100",
			height: clienttypes.NewHeight(0, 100),
		},
		{
			name:   "revision 1, height 200",
			height: clienttypes.NewHeight(1, 200),
		},
		{
			name:   "large height",
			height: clienttypes.NewHeight(5, 1000000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			iterKey := IterationKey(tc.height)
			recoveredHeight := GetHeightFromIterationKey(iterKey)
			require.Equal(t, tc.height.GetRevisionNumber(), recoveredHeight.GetRevisionNumber())
			require.Equal(t, tc.height.GetRevisionHeight(), recoveredHeight.GetRevisionHeight())
		})
	}
}

func TestSetAndGetIterationKey(t *testing.T) {
	clientStore := setupClientStore(t)
	height := clienttypes.NewHeight(1, 100)

	// Initially no iteration key
	key := GetIterationKey(clientStore, height)
	require.Nil(t, key)

	// Set iteration key
	SetIterationKey(clientStore, height)

	// Get iteration key
	key = GetIterationKey(clientStore, height)
	require.NotNil(t, key)
}

func TestIterateConsensusStateAscending(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()

	// Set multiple consensus states
	heights := []clienttypes.Height{
		clienttypes.NewHeight(1, 100),
		clienttypes.NewHeight(1, 200),
		clienttypes.NewHeight(1, 300),
	}

	for _, h := range heights {
		cs := createTestConsensusState(time.Now().UTC())
		setConsensusState(clientStore, cdc, cs, h)
		SetIterationKey(clientStore, h)
	}

	// Iterate and verify order
	var iteratedHeights []uint64
	IterateConsensusStateAscending(clientStore, func(height exported.Height) bool {
		iteratedHeights = append(iteratedHeights, height.GetRevisionHeight())
		return false // continue iteration
	})

	require.Len(t, iteratedHeights, 3)
	require.Equal(t, uint64(100), iteratedHeights[0])
	require.Equal(t, uint64(200), iteratedHeights[1])
	require.Equal(t, uint64(300), iteratedHeights[2])
}

func TestIterateConsensusStateAscending_StopEarly(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()

	// Set multiple consensus states
	heights := []clienttypes.Height{
		clienttypes.NewHeight(1, 100),
		clienttypes.NewHeight(1, 200),
		clienttypes.NewHeight(1, 300),
	}

	for _, h := range heights {
		cs := createTestConsensusState(time.Now().UTC())
		setConsensusState(clientStore, cdc, cs, h)
		SetIterationKey(clientStore, h)
	}

	// Iterate but stop after first
	var iteratedHeights []uint64
	IterateConsensusStateAscending(clientStore, func(height exported.Height) bool {
		iteratedHeights = append(iteratedHeights, height.GetRevisionHeight())
		return true // stop iteration
	})

	require.Len(t, iteratedHeights, 1)
	require.Equal(t, uint64(100), iteratedHeights[0])
}

func TestGetNextConsensusState(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()

	// Set multiple consensus states
	heights := []clienttypes.Height{
		clienttypes.NewHeight(1, 100),
		clienttypes.NewHeight(1, 200),
		clienttypes.NewHeight(1, 300),
	}

	for _, h := range heights {
		cs := createTestConsensusState(time.Now().UTC())
		setConsensusState(clientStore, cdc, cs, h)
		SetIterationKey(clientStore, h)
	}

	// Get next from height 100 - should be 200
	nextCS, found := GetNextConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 100))
	require.True(t, found)
	require.NotNil(t, nextCS)

	// Get next from height 300 - should not exist
	_, found = GetNextConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 300))
	require.False(t, found)

	// Get next from height 150 (doesn't exist) - should be 200
	nextCS, found = GetNextConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 150))
	require.True(t, found)
	require.NotNil(t, nextCS)
}

func TestGetPreviousConsensusState(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()

	// Set multiple consensus states
	heights := []clienttypes.Height{
		clienttypes.NewHeight(1, 100),
		clienttypes.NewHeight(1, 200),
		clienttypes.NewHeight(1, 300),
	}

	for _, h := range heights {
		cs := createTestConsensusState(time.Now().UTC())
		setConsensusState(clientStore, cdc, cs, h)
		SetIterationKey(clientStore, h)
	}

	// Get previous from height 300 - should be 200
	prevCS, found := GetPreviousConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 300))
	require.True(t, found)
	require.NotNil(t, prevCS)

	// Get previous from height 100 - should not exist
	_, found = GetPreviousConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 100))
	require.False(t, found)

	// Get previous from height 250 (doesn't exist) - should be 200
	prevCS, found = GetPreviousConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 250))
	require.True(t, found)
	require.NotNil(t, prevCS)
}

func TestProcessedTimeKey(t *testing.T) {
	height := clienttypes.NewHeight(1, 100)
	key := ProcessedTimeKey(height)
	require.NotEmpty(t, key)
	require.Contains(t, string(key), "processedTime")
}

func TestProcessedHeightKey(t *testing.T) {
	height := clienttypes.NewHeight(1, 100)
	key := ProcessedHeightKey(height)
	require.NotEmpty(t, key)
	require.Contains(t, string(key), "processedHeight")
}

func TestPruneAllExpiredConsensusStates(t *testing.T) {
	clientStore := setupClientStore(t)
	cdc := getTestCodec()

	clientState := createTestClientState(testChainID, clienttypes.NewHeight(1, 300), false)

	// Create expired consensus state (time in the past beyond trusting period)
	expiredTime := time.Now().UTC().Add(-testTrustingPeriod - time.Hour)
	expiredCS := createTestConsensusState(expiredTime)
	setConsensusState(clientStore, cdc, expiredCS, clienttypes.NewHeight(1, 100))
	SetIterationKey(clientStore, clienttypes.NewHeight(1, 100))

	// Create non-expired consensus state
	currentTime := time.Now().UTC()
	validCS := createTestConsensusState(currentTime)
	setConsensusState(clientStore, cdc, validCS, clienttypes.NewHeight(1, 200))
	SetIterationKey(clientStore, clienttypes.NewHeight(1, 200))

	ctx := getTestContext(t, time.Now().UTC())

	// Prune expired states
	numPruned := PruneAllExpiredConsensusStates(ctx, clientStore, cdc, clientState)
	require.Equal(t, 1, numPruned)

	// Verify expired state is gone
	_, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 100))
	require.False(t, found)

	// Verify valid state is still there
	_, found = GetConsensusState(clientStore, cdc, clienttypes.NewHeight(1, 200))
	require.True(t, found)
}
