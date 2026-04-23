package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
)

func TestConstructUpgradeClientMerklePath(t *testing.T) {
	upgradePath := []string{"upgrade", "upgradedIBCState"}
	height := clienttypes.NewHeight(1, 100)

	path := constructUpgradeClientMerklePath(upgradePath, height)

	require.NotEmpty(t, path.GetKeyPath())
	// Should contain the upgrade path elements plus the height and client key
	keyPath := path.GetKeyPath()
	require.Len(t, keyPath, 2)
	require.Contains(t, string(keyPath[1]), "100")
	require.Contains(t, string(keyPath[1]), "upgradedClient")
}

func TestConstructUpgradeConsStateMerklePath(t *testing.T) {
	upgradePath := []string{"upgrade", "upgradedIBCState"}
	height := clienttypes.NewHeight(1, 100)

	path := constructUpgradeConsStateMerklePath(upgradePath, height)

	require.NotEmpty(t, path.GetKeyPath())
	// Should contain the upgrade path elements plus the height and consensus state key
	keyPath := path.GetKeyPath()
	require.Len(t, keyPath, 2)
	require.Contains(t, string(keyPath[1]), "100")
	require.Contains(t, string(keyPath[1]), "upgradedConsState")
}

func TestCalculateNewTrustingPeriod(t *testing.T) {
	testCases := []struct {
		name                string
		trustingPeriod      time.Duration
		originalUnbonding   time.Duration
		newUnbonding        time.Duration
		expectedTrustPeriod time.Duration
	}{
		{
			name:                "unbonding period halved",
			trustingPeriod:      time.Hour * 24 * 14, // 14 days
			originalUnbonding:   time.Hour * 24 * 21, // 21 days
			newUnbonding:        time.Hour * 24 * 10, // ~10 days (about half)
			expectedTrustPeriod: time.Hour * 24 * 14 * 10 / 21, // roughly 6.67 days
		},
		{
			name:                "unbonding period unchanged",
			trustingPeriod:      time.Hour * 24 * 14,
			originalUnbonding:   time.Hour * 24 * 21,
			newUnbonding:        time.Hour * 24 * 21,
			expectedTrustPeriod: time.Hour * 24 * 14, // same as original
		},
		{
			name:                "small values",
			trustingPeriod:      time.Hour * 10,
			originalUnbonding:   time.Hour * 20,
			newUnbonding:        time.Hour * 10,
			expectedTrustPeriod: time.Hour * 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateNewTrustingPeriod(tc.trustingPeriod, tc.originalUnbonding, tc.newUnbonding)
			require.Equal(t, tc.expectedTrustPeriod, result)
		})
	}
}

func TestConstructUpgradeClientMerklePath_SingleElement(t *testing.T) {
	upgradePath := []string{"singlePath"}
	height := clienttypes.NewHeight(0, 50)

	path := constructUpgradeClientMerklePath(upgradePath, height)

	keyPath := path.GetKeyPath()
	// With single element, result should have one element with appended keys
	require.Len(t, keyPath, 1)
	require.Contains(t, string(keyPath[0]), "50")
	require.Contains(t, string(keyPath[0]), "upgradedClient")
}

func TestConstructUpgradeClientMerklePath_MultipleElements(t *testing.T) {
	upgradePath := []string{"upgrade", "ibc", "state"}
	height := clienttypes.NewHeight(2, 150)

	path := constructUpgradeClientMerklePath(upgradePath, height)

	keyPath := path.GetKeyPath()
	// Should have 3 elements (upgrade, ibc, and the appended key)
	require.Len(t, keyPath, 3)
	require.Equal(t, "upgrade", string(keyPath[0]))
	require.Equal(t, "ibc", string(keyPath[1]))
	require.Contains(t, string(keyPath[2]), "150")
}

func TestMerklePath_NotNil(t *testing.T) {
	upgradePath := []string{"upgrade", "upgradedIBCState"}
	height := clienttypes.NewHeight(1, 100)

	clientPath := constructUpgradeClientMerklePath(upgradePath, height)
	consStatePath := constructUpgradeConsStateMerklePath(upgradePath, height)

	// Ensure the paths are properly formed MerklePaths
	require.IsType(t, commitmenttypesv2.MerklePath{}, clientPath)
	require.IsType(t, commitmenttypesv2.MerklePath{}, consStatePath)
}
