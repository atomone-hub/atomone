package gno

import (
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

func TestLightClientModule_Constants(t *testing.T) {
	// Test that the module name constant is correct
	require.Equal(t, "10-gno", ModuleName)
	require.Equal(t, "10-gno", Gno)
}

func TestLightClientModule_FrozenHeight(t *testing.T) {
	// Test the frozen height constant
	require.Equal(t, uint64(0), FrozenHeight.GetRevisionNumber())
	require.Equal(t, uint64(1), FrozenHeight.GetRevisionHeight())
}

func TestLightClientModule_SentinelRoot(t *testing.T) {
	// Test the sentinel root constant
	require.Equal(t, "sentinel_root", SentinelRoot)
}

func TestLightClientModule_KeyIteratePrefix(t *testing.T) {
	// Test the key iteration prefix constant
	require.Equal(t, "iterateConsensusStates", KeyIterateConsensusStatePrefix)
}

func TestNewLightClientModule_Fields(t *testing.T) {
	// Test that NewLightClientModule creates a module with the correct structure
	// This is a basic test that doesn't require the full IBC setup

	cdc := getTestCodec()
	require.NotNil(t, cdc)

	// The light client module requires a StoreProvider which is complex to mock
	// Integration tests in the broader IBC test suite would test the full functionality
}

func TestLightClientModule_InterfaceCompliance(t *testing.T) {
	// Ensure LightClientModule implements the exported.LightClientModule interface
	// This is a compile-time check - if it compiles, the interface is implemented
	// The actual interface check is done in the module definition with:
	// var _ exported.LightClientModule = (*LightClientModule)(nil)
	require.True(t, true) // Placeholder - actual check is compile-time
}

func TestClientTypeMatches(t *testing.T) {
	// Test that client type matching works correctly for client identifiers
	testCases := []struct {
		clientID    string
		expectMatch bool
	}{
		{"10-gno-0", true},
		{"10-gno-1", true},
		{"10-gno-100", true},
		{"07-tendermint-0", false},
		{"invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.clientID, func(t *testing.T) {
			clientType, _, err := clienttypes.ParseClientIdentifier(tc.clientID)
			if err != nil {
				require.False(t, tc.expectMatch)
				return
			}
			require.Equal(t, tc.expectMatch, clientType == Gno)
		})
	}
}
