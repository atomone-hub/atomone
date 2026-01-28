package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
)

func TestNewConsensusState(t *testing.T) {
	timestamp := time.Now().UTC()
	root := commitmenttypes.NewMerkleRoot([]byte("test-app-hash"))
	nextValsHash := make([]byte, 32)

	cs := NewConsensusState(timestamp, root, nextValsHash)

	require.Equal(t, timestamp, cs.Timestamp)
	require.Equal(t, root, cs.Root)
	require.Equal(t, nextValsHash, cs.NextValidatorsHash)
	require.Equal(t, Gno, cs.LcType)
}

func TestConsensusState_ClientType(t *testing.T) {
	cs := &ConsensusState{}
	require.Equal(t, Gno, cs.ClientType())
}

func TestConsensusState_GetRoot(t *testing.T) {
	root := commitmenttypes.NewMerkleRoot([]byte("test-root"))
	cs := &ConsensusState{
		Root: root,
	}

	require.Equal(t, root, cs.GetRoot())
}

func TestConsensusState_GetTimestamp(t *testing.T) {
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	cs := &ConsensusState{
		Timestamp: timestamp,
	}

	require.Equal(t, uint64(timestamp.UnixNano()), cs.GetTimestamp())
}

func TestConsensusState_GetTimestamp_ZeroTime(t *testing.T) {
	cs := &ConsensusState{
		Timestamp: time.Time{},
	}

	// Unix time of zero time is negative
	expected := uint64(time.Time{}.UnixNano())
	require.Equal(t, expected, cs.GetTimestamp())
}

func TestConsensusState_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name        string
		consState   func() *ConsensusState
		expectErr   bool
		errContains string
	}{
		{
			name: "valid consensus state",
			consState: func() *ConsensusState {
				return createTestConsensusState(time.Now().UTC())
			},
			expectErr: false,
		},
		{
			name: "empty root",
			consState: func() *ConsensusState {
				cs := createTestConsensusState(time.Now().UTC())
				cs.Root = commitmenttypes.MerkleRoot{}
				return cs
			},
			expectErr:   true,
			errContains: "root cannot be empty",
		},
		{
			name: "invalid next validators hash - wrong length",
			consState: func() *ConsensusState {
				cs := createTestConsensusState(time.Now().UTC())
				cs.NextValidatorsHash = []byte("short")
				return cs
			},
			expectErr:   true,
			errContains: "next validators hash is invalid",
		},
		{
			name: "timestamp at unix zero",
			consState: func() *ConsensusState {
				cs := createTestConsensusState(time.Unix(0, 0))
				return cs
			},
			expectErr:   true,
			errContains: "timestamp must be a positive Unix time",
		},
		{
			name: "timestamp before unix epoch",
			consState: func() *ConsensusState {
				cs := createTestConsensusState(time.Unix(-100, 0))
				return cs
			},
			expectErr:   true,
			errContains: "timestamp must be a positive Unix time",
		},
		{
			name: "timestamp at positive unix time",
			consState: func() *ConsensusState {
				cs := createTestConsensusState(time.Unix(1, 0))
				return cs
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs := tc.consState()
			err := cs.ValidateBasic()

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConsensusState_Sentinel(t *testing.T) {
	require.Equal(t, "sentinel_root", SentinelRoot)
}
