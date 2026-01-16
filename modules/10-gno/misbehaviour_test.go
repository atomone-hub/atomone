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
		name        string
		misbehaviour func() *Misbehaviour
		expectErr   bool
		errContains string
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

func TestFrozenHeight(t *testing.T) {
	require.Equal(t, clienttypes.NewHeight(0, 1), FrozenHeight)
}
