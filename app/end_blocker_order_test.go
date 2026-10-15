package atomone

import (
	"testing"

	"github.com/stretchr/testify/require"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
)

// TestCoreDAOsEndBlockerRunsBeforeGov guards the ordering invariant that the
// x/coredaos EndBlocker runs before x/gov's. VetoProposal defers the deletion of
// a vetoed proposal's votes to the coredaos EndBlocker; if gov's EndBlocker ran
// first it could observe those votes during its quorum check, pass quorum, and
// re-insert the vetoed proposal into the ActiveProposalsQueue.
func TestCoreDAOsEndBlockerRunsBeforeGov(t *testing.T) {
	order := orderEndBlockers()

	indexOf := func(name string) int {
		for i, n := range order {
			if n == name {
				return i
			}
		}
		return -1
	}

	coredaosIdx := indexOf(coredaostypes.ModuleName)
	govIdx := indexOf(govtypes.ModuleName)

	require.NotEqual(t, -1, coredaosIdx, "coredaos module missing from end blockers")
	require.NotEqual(t, -1, govIdx, "gov module missing from end blockers")
	require.Less(t, coredaosIdx, govIdx,
		"x/coredaos EndBlocker must run before x/gov EndBlocker to avoid resurrecting vetoed proposals")
}
