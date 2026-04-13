package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ics23 "github.com/cosmos/ics23/go"
)

func TestNewClientState(t *testing.T) {
	chainID := testChainID
	height := clienttypes.NewHeight(1, 100)

	cs := NewClientState(
		chainID,
		DefaultTrustLevel,
		testTrustingPeriod,
		testUnbondingPeriod,
		testMaxClockDrift,
		height,
		[]*ics23.ProofSpec{ics23.IavlSpec, ics23.TendermintSpec},
		[]string{"upgrade", "upgradedIBCState"},
	)

	require.Equal(t, chainID, cs.ChainId)
	require.Equal(t, DefaultTrustLevel, cs.TrustLevel)
	require.Equal(t, testTrustingPeriod, cs.TrustingPeriod)
	require.Equal(t, testUnbondingPeriod, cs.UnbondingPeriod)
	require.Equal(t, testMaxClockDrift, cs.MaxClockDrift)
	require.Equal(t, height, cs.LatestHeight)
	require.True(t, cs.FrozenHeight.IsZero())
	require.Len(t, cs.ProofSpecs, 2)
	require.Len(t, cs.UpgradePath, 2)
}

func TestClientState_GetChainID(t *testing.T) {
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	require.Equal(t, testChainID, cs.GetChainID())
}

func TestClientState_ClientType(t *testing.T) {
	cs := &ClientState{}
	require.Equal(t, Gno, cs.ClientType())
}

func TestClientState_IsExpired(t *testing.T) {
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)

	testCases := []struct {
		name            string
		latestTimestamp time.Time
		now             time.Time
		expired         bool
	}{
		{
			name:            "not expired - well within trusting period",
			latestTimestamp: time.Now().UTC(),
			now:             time.Now().UTC().Add(time.Hour),
			expired:         false,
		},
		{
			name:            "not expired - at boundary (still within)",
			latestTimestamp: time.Now().UTC(),
			now:             time.Now().UTC().Add(testTrustingPeriod - time.Second),
			expired:         false,
		},
		{
			name:            "expired - exactly at trusting period",
			latestTimestamp: time.Now().UTC(),
			now:             time.Now().UTC().Add(testTrustingPeriod),
			expired:         true,
		},
		{
			name:            "expired - past trusting period",
			latestTimestamp: time.Now().UTC(),
			now:             time.Now().UTC().Add(testTrustingPeriod + time.Hour),
			expired:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cs.IsExpired(tc.latestTimestamp, tc.now)
			require.Equal(t, tc.expired, result)
		})
	}
}

func TestClientState_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		clientState func() *ClientState
		expectErr   bool
		errMsg      string
	}{
		{
			name: "valid client state",
			clientState: func() *ClientState {
				return createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
			},
			expectErr: false,
		},
		{
			name: "empty chain ID",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.ChainId = ""
				return cs
			},
			expectErr: true,
			errMsg:    "chain id cannot be empty string",
		},
		{
			name: "chain ID with spaces only",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.ChainId = "   "
				return cs
			},
			expectErr: true,
			errMsg:    "chain id cannot be empty string",
		},
		{
			name: "invalid trust level - zero denominator",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustLevel = Fraction{Numerator: 1, Denominator: 0}
				return cs
			},
			expectErr: true,
			errMsg:    "invalid trust level",
		},
		{
			name: "invalid trust level - too low",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustLevel = Fraction{Numerator: 1, Denominator: 4} // 1/4 < 1/3
				return cs
			},
			expectErr: true,
			errMsg:    "invalid trust level",
		},
		{
			name: "invalid trust level - numerator > denominator",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustLevel = Fraction{Numerator: 2, Denominator: 1}
				return cs
			},
			expectErr: true,
			errMsg:    "invalid trust level",
		},
		{
			name: "zero trusting period",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustingPeriod = 0
				return cs
			},
			expectErr: true,
			errMsg:    "trusting period must be greater than zero",
		},
		{
			name: "negative trusting period",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustingPeriod = -time.Hour
				return cs
			},
			expectErr: true,
			errMsg:    "trusting period must be greater than zero",
		},
		{
			name: "zero unbonding period",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.UnbondingPeriod = 0
				return cs
			},
			expectErr: true,
			errMsg:    "unbonding period must be greater than zero",
		},
		{
			name: "zero max clock drift",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.MaxClockDrift = 0
				return cs
			},
			expectErr: true,
			errMsg:    "max clock drift must be greater than zero",
		},
		{
			name: "trusting period >= unbonding period",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustingPeriod = testUnbondingPeriod // Equal
				return cs
			},
			expectErr: true,
			errMsg:    "trusting period",
		},
		{
			name: "trusting period > unbonding period",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustingPeriod = testUnbondingPeriod + time.Hour
				return cs
			},
			expectErr: true,
			errMsg:    "trusting period",
		},
		{
			name: "latest height revision height is zero",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 0), false)
				return cs
			},
			expectErr: true,
			errMsg:    "revision height cannot be zero",
		},
		{
			name: "revision number mismatch with chain ID",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(5, 100), false) // Revision 5 doesn't match chain ID "gno-test-1"
				return cs
			},
			expectErr: true,
			errMsg:    "revision number must match chain id revision number",
		},
		{
			name: "nil proof specs",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.ProofSpecs = nil
				return cs
			},
			expectErr: true,
			errMsg:    "proof specs cannot be nil",
		},
		{
			name: "nil proof spec in slice",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.ProofSpecs = []*ics23.ProofSpec{nil}
				return cs
			},
			expectErr: true,
			errMsg:    "proof spec cannot be nil at index",
		},
		{
			name: "empty string in upgrade path",
			clientState: func() *ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.UpgradePath = []string{"upgrade", ""}
				return cs
			},
			expectErr: true,
			errMsg:    "key in upgrade path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs := tc.clientState()
			err := cs.Validate()

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClientState_ZeroCustomFields(t *testing.T) {
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	cs.TrustLevel = Fraction{Numerator: 2, Denominator: 3}
	cs.TrustingPeriod = time.Hour * 100
	cs.MaxClockDrift = time.Minute * 5

	zeroed := cs.ZeroCustomFields()

	// These fields should be preserved (chain-specific)
	require.Equal(t, cs.ChainId, zeroed.ChainId)
	require.Equal(t, cs.UnbondingPeriod, zeroed.UnbondingPeriod)
	require.Equal(t, cs.LatestHeight, zeroed.LatestHeight)
	require.Equal(t, cs.ProofSpecs, zeroed.ProofSpecs)
	require.Equal(t, cs.UpgradePath, zeroed.UpgradePath)

	// These fields should be zeroed (client-specific)
	require.Equal(t, Fraction{}, zeroed.TrustLevel)
	require.Equal(t, time.Duration(0), zeroed.TrustingPeriod)
	require.Equal(t, time.Duration(0), zeroed.MaxClockDrift)
}

func TestClientState_Status(t *testing.T) {
	testCases := []struct {
		name           string
		setupState     func() (*ClientState, *ConsensusState, time.Time)
		expectedStatus exported.Status
	}{
		{
			name: "active client",
			setupState: func() (*ClientState, *ConsensusState, time.Time) {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				consState := createTestConsensusState(time.Now().UTC())
				blockTime := time.Now().UTC().Add(time.Hour) // 1 hour after consensus state
				return cs, consState, blockTime
			},
			expectedStatus: exported.Active,
		},
		{
			name: "frozen client",
			setupState: func() (*ClientState, *ConsensusState, time.Time) {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), true) // frozen
				consState := createTestConsensusState(time.Now().UTC())
				blockTime := time.Now().UTC().Add(time.Hour)
				return cs, consState, blockTime
			},
			expectedStatus: exported.Frozen,
		},
		{
			name: "expired client - no consensus state",
			setupState: func() (*ClientState, *ConsensusState, time.Time) {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				// Return nil consensus state - client store will have no consensus state at latest height
				blockTime := time.Now().UTC()
				return cs, nil, blockTime
			},
			expectedStatus: exported.Expired,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, consState, blockTime := tc.setupState()
			clientStore := setupClientStore(t)
			cdc := getTestCodec()
			ctx := getTestContext(t, blockTime)

			if consState != nil {
				setConsensusState(clientStore, cdc, consState, cs.LatestHeight)
			}

			status := cs.status(ctx, clientStore, cdc)
			require.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestClientState_GetTimestampAtHeight(t *testing.T) {
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	clientStore := setupClientStore(t)
	cdc := getTestCodec()

	// Test with existing consensus state
	consState := createTestConsensusState(time.Now().UTC())
	setConsensusState(clientStore, cdc, consState, cs.LatestHeight)

	timestamp, err := cs.getTimestampAtHeight(clientStore, cdc, cs.LatestHeight)
	require.NoError(t, err)
	require.Equal(t, consState.GetTimestamp(), timestamp)

	// Test with non-existent consensus state
	_, err = cs.getTimestampAtHeight(clientStore, cdc, clienttypes.NewHeight(1, 200))
	require.Error(t, err)
	require.Contains(t, err.Error(), "consensus state not found")
}

func TestClientState_Initialize(t *testing.T) {
	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
	clientStore := setupClientStore(t)
	cdc := getTestCodec()
	ctx := getTestContext(t, time.Now().UTC())

	// Test with valid consensus state
	consState := createTestConsensusState(time.Now().UTC())
	err := cs.initialize(ctx, cdc, clientStore, consState)
	require.NoError(t, err)

	// Verify client state was stored
	storedCS, found := getClientState(clientStore, cdc)
	require.True(t, found)
	require.Equal(t, cs.ChainId, storedCS.ChainId)

	// Verify consensus state was stored
	storedConsState, found := GetConsensusState(clientStore, cdc, cs.LatestHeight)
	require.True(t, found)
	require.Equal(t, consState.Timestamp, storedConsState.Timestamp)
}

// Test that initialize fails with wrong consensus state type - skip this test
// since creating a mock that implements the full interface requires proto codegen.
// The validation of consensus state type is tested through integration tests.
