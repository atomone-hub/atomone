package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ics23 "github.com/cosmos/ics23/go"
)

func TestIsMatchingClientState(t *testing.T) {
	testCases := []struct {
		name      string
		subject   func() ClientState
		substitute func() ClientState
		isMatching bool
	}{
		{
			name: "matching client states - all parameters same",
			subject: func() ClientState {
				return *createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
			},
			substitute: func() ClientState {
				return *createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
			},
			isMatching: true,
		},
		{
			name: "matching client states - different latest heights",
			subject: func() ClientState {
				return *createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
			},
			substitute: func() ClientState {
				return *createTestClientState(testChainID, clienttypes.NewHeight(1, 200), false)
			},
			isMatching: true,
		},
		{
			name: "matching client states - different frozen heights",
			subject: func() ClientState {
				return *createTestClientState(testChainID, clienttypes.NewHeight(1, 100), true)
			},
			substitute: func() ClientState {
				return *createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
			},
			isMatching: true,
		},
		{
			name: "matching client states - different trusting periods",
			subject: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustingPeriod = time.Hour * 24 * 7
				return *cs
			},
			substitute: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.TrustingPeriod = time.Hour * 24 * 14
				return *cs
			},
			isMatching: true,
		},
		{
			name: "matching client states - different chain IDs",
			subject: func() ClientState {
				return *createTestClientState("gno-test-1", clienttypes.NewHeight(1, 100), false)
			},
			substitute: func() ClientState {
				return *createTestClientState("gno-test-2", clienttypes.NewHeight(2, 100), false)
			},
			isMatching: true, // Chain ID is zeroed for comparison
		},
		{
			name: "not matching - different unbonding periods",
			subject: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.UnbondingPeriod = time.Hour * 24 * 21
				return *cs
			},
			substitute: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.UnbondingPeriod = time.Hour * 24 * 28
				return *cs
			},
			isMatching: false,
		},
		{
			name: "not matching - different max clock drift",
			subject: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.MaxClockDrift = time.Second * 10
				return *cs
			},
			substitute: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.MaxClockDrift = time.Second * 20
				return *cs
			},
			isMatching: false,
		},
		{
			name: "not matching - different proof specs",
			subject: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.ProofSpecs = []*ics23.ProofSpec{ics23.IavlSpec}
				return *cs
			},
			substitute: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.ProofSpecs = []*ics23.ProofSpec{ics23.IavlSpec, ics23.TendermintSpec}
				return *cs
			},
			isMatching: false,
		},
		{
			name: "not matching - different upgrade paths",
			subject: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.UpgradePath = []string{"upgrade", "path1"}
				return *cs
			},
			substitute: func() ClientState {
				cs := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				cs.UpgradePath = []string{"upgrade", "path2"}
				return *cs
			},
			isMatching: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subject := tc.subject()
			substitute := tc.substitute()
			result := IsMatchingClientState(subject, substitute)
			require.Equal(t, tc.isMatching, result)
		})
	}
}

func TestCheckSubstituteAndUpdateState(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func() (*ClientState, *ClientState, *ConsensusState)
		expectErr bool
		errContains string
	}{
		{
			name: "successful substitution - unfreezes frozen client",
			setup: func() (*ClientState, *ClientState, *ConsensusState) {
				subject := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), true) // frozen
				substitute := createTestClientState(testChainID, clienttypes.NewHeight(1, 200), false)
				consState := createTestConsensusState(time.Now().UTC())
				return subject, substitute, consState
			},
			expectErr: false,
		},
		{
			name: "successful substitution - active client",
			setup: func() (*ClientState, *ClientState, *ConsensusState) {
				subject := createTestClientState(testChainID, clienttypes.NewHeight(1, 100), false)
				substitute := createTestClientState(testChainID, clienttypes.NewHeight(1, 200), false)
				consState := createTestConsensusState(time.Now().UTC())
				return subject, substitute, consState
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subject, substitute, consState := tc.setup()

			// Setup stores
			subjectStore := setupClientStore(t)
			substituteStore := setupClientStore(t)
			cdc := getTestCodec()
			ctx := getTestContext(t, time.Now().UTC())

			// Set up substitute client store with consensus state and metadata
			setConsensusState(substituteStore, cdc, consState, substitute.LatestHeight)
			SetProcessedTime(substituteStore, substitute.LatestHeight, uint64(time.Now().UnixNano()))
			SetProcessedHeight(substituteStore, substitute.LatestHeight, clienttypes.NewHeight(0, 1))

			// Run the function
			err := subject.CheckSubstituteAndUpdateState(ctx, cdc, subjectStore, substituteStore, substitute)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)

				// Verify the subject client state was updated
				updatedCS, found := getClientState(subjectStore, cdc)
				require.True(t, found)
				require.Equal(t, substitute.LatestHeight, updatedCS.LatestHeight)
				require.True(t, updatedCS.FrozenHeight.IsZero()) // Should be unfrozen
			}
		})
	}
}
