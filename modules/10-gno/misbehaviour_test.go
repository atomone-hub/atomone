package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

func TestNewMisbehaviour(t *testing.T) {
	clientID := testClientID
	header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
	header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())

	misbehaviour := NewMisbehaviour(clientID, header1, header2)

	require.Equal(t, clientID, misbehaviour.ClientId)
	require.Equal(t, header1, misbehaviour.Header1)
	require.Equal(t, header2, misbehaviour.Header2)
}

func TestMisbehaviour_ClientType(t *testing.T) {
	misbehaviour := &Misbehaviour{}
	require.Equal(t, Gno, misbehaviour.ClientType())
}

func TestMisbehaviour_GetTime(t *testing.T) {
	time1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)

	testCases := []struct {
		name         string
		header1Time  time.Time
		header2Time  time.Time
		expectedTime time.Time
	}{
		{
			name:         "header1 time is after header2 time",
			header1Time:  time2,
			header2Time:  time1,
			expectedTime: time2,
		},
		{
			name:         "header2 time is after header1 time",
			header1Time:  time1,
			header2Time:  time2,
			expectedTime: time2,
		},
		{
			name:         "both headers have same time",
			header1Time:  time1,
			header2Time:  time1,
			expectedTime: time1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), tc.header1Time)
			header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), tc.header2Time)

			misbehaviour := NewMisbehaviour(testClientID, header1, header2)
			require.Equal(t, tc.expectedTime, misbehaviour.GetTime())
		})
	}
}

func TestMisbehaviour_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name         string
		misbehaviour func() *Misbehaviour
		expectErr    bool
		errContains  string
	}{
		{
			name: "nil header1",
			misbehaviour: func() *Misbehaviour {
				header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				return NewMisbehaviour(testClientID, nil, header2)
			},
			expectErr:   true,
			errContains: "Header1 cannot be nil",
		},
		{
			name: "nil header2",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				return NewMisbehaviour(testClientID, header1, nil)
			},
			expectErr:   true,
			errContains: "Header2 cannot be nil",
		},
		{
			name: "header1 trusted height revision height is zero",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 0), time.Now().UTC())
				header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				return NewMisbehaviour(testClientID, header1, header2)
			},
			expectErr:   true,
			errContains: "Header1 cannot have zero revision height",
		},
		{
			name: "header2 trusted height revision height is zero",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 0), time.Now().UTC())
				return NewMisbehaviour(testClientID, header1, header2)
			},
			expectErr:   true,
			errContains: "Header2 cannot have zero revision height",
		},
		{
			name: "nil trusted validators in header1",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				header1.TrustedValidators = nil
				header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				return NewMisbehaviour(testClientID, header1, header2)
			},
			expectErr:   true,
			errContains: "trusted validator set in Header1 cannot be empty",
		},
		{
			name: "nil trusted validators in header2",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				header2.TrustedValidators = nil
				return NewMisbehaviour(testClientID, header1, header2)
			},
			expectErr:   true,
			errContains: "trusted validator set in Header2 cannot be empty",
		},
		{
			name: "chain IDs don't match",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, "gno-chain-1", 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				header2 := createTestHeader(t, "gno-chain-2", 100, clienttypes.NewHeight(2, 50), time.Now().UTC())
				return NewMisbehaviour(testClientID, header1, header2)
			},
			expectErr:   true,
			errContains: "headers must have identical chainIDs",
		},
		{
			name: "invalid client identifier",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				return NewMisbehaviour("invalid!client", header1, header2)
			},
			expectErr:   true,
			errContains: "misbehaviour client ID is invalid",
		},
		{
			name: "header1 height less than header2 height - causes validation error",
			misbehaviour: func() *Misbehaviour {
				header1 := createTestHeader(t, testChainID, 50, clienttypes.NewHeight(1, 25), time.Now().UTC())
				header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), time.Now().UTC())
				return NewMisbehaviour(testClientID, header1, header2)
			},
			expectErr:   true,
			errContains: "Header1 height is less than Header2 height", // Misbehaviour requires h1 >= h2
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			misbehaviour := tc.misbehaviour()
			err := misbehaviour.ValidateBasic()

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMisbehaviour_ValidateBasic_CommitBlockID tests that ValidateBasic uses
// the commit's BlockId (the block actually committed to) rather than the
// header's LastBlockId (the previous block) when verifying commits in
// misbehaviour. This was the bug fixed in Oak Issue #2.
//
// With the buggy code, validCommit was called with Header.LastBlockId,
// causing VerifyCommit to always fail because signatures were verified
// against the wrong block ID. This meant all valid misbehaviour was rejected.
func TestMisbehaviour_ValidateBasic_CommitBlockID(t *testing.T) {
	blockTime := time.Now().UTC()

	// Create two independently valid headers at the same height (fork).
	header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), blockTime)
	header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), blockTime.Add(time.Second))

	// Sanity: the two headers have different commit block IDs
	require.NotEqual(t,
		header1.SignedHeader.Commit.BlockId.Hash,
		header2.SignedHeader.Commit.BlockId.Hash,
		"test setup: two independent headers should have different commit block ID hashes",
	)

	// Sanity: LastBlockId differs from Commit.BlockId (parent vs current block)
	require.NotEqual(t,
		header1.SignedHeader.Header.LastBlockId.Hash,
		header1.SignedHeader.Commit.BlockId.Hash,
		"test setup: LastBlockId and Commit.BlockId should differ",
	)

	misbehaviour := NewMisbehaviour(testClientID, header1, header2)
	err := misbehaviour.ValidateBasic()
	require.NoError(t, err, "valid fork misbehaviour should pass ValidateBasic; "+
		"if this fails, validCommit may be using Header.LastBlockId instead of Commit.BlockId")
}

// TestCheckForMisbehaviour_ForkDetection tests that CheckForMisbehaviour
// correctly detects forks by comparing commit block IDs (not header LastBlockIds).
// This was the bug fixed in Oak Issue #2.
func TestCheckForMisbehaviour_ForkDetection(t *testing.T) {
	blockTime := time.Now().UTC()

	// Create two headers at the same height with different commits (a fork)
	header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), blockTime)
	header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), blockTime.Add(time.Second))

	// Ensure commit block IDs differ (fork condition)
	require.NotEqual(t,
		header1.SignedHeader.Commit.BlockId.Hash,
		header2.SignedHeader.Commit.BlockId.Hash,
	)

	// Make both headers share the same LastBlockId.
	// The old buggy code used LastBlockId for comparison, so it would have
	// seen equal hashes and missed the fork.
	header1.SignedHeader.Header.LastBlockId = header2.SignedHeader.Header.LastBlockId

	misbehaviour := NewMisbehaviour(testClientID, header1, header2)

	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 99), false)
	result := cs.CheckForMisbehaviour(
		getTestContext(t, blockTime),
		getTestCodec(),
		setupClientStore(t),
		misbehaviour,
	)
	require.True(t, result, "CheckForMisbehaviour should detect fork when commit block IDs differ, even if LastBlockIds are the same")
}

// TestCheckForMisbehaviour_NoForkSameCommit tests that CheckForMisbehaviour
// returns false when commit block IDs are identical (no fork).
func TestCheckForMisbehaviour_NoForkSameCommit(t *testing.T) {
	blockTime := time.Now().UTC()

	// Create the same header twice (identical commits)
	header1 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), blockTime)

	// Deep copy header1 as header2 — same commit block ID
	header2 := createTestHeader(t, testChainID, 100, clienttypes.NewHeight(1, 50), blockTime)
	header2.SignedHeader.Commit.BlockId = header1.SignedHeader.Commit.BlockId
	header2.SignedHeader.Header = header1.SignedHeader.Header

	misbehaviour := NewMisbehaviour(testClientID, header1, header2)

	cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 99), false)
	result := cs.CheckForMisbehaviour(
		getTestContext(t, blockTime),
		getTestCodec(),
		setupClientStore(t),
		misbehaviour,
	)
	require.False(t, result, "CheckForMisbehaviour should not detect fork when commit block IDs are identical")
}

func TestFrozenHeight(t *testing.T) {
	require.Equal(t, clienttypes.NewHeight(0, 1), FrozenHeight)
}
