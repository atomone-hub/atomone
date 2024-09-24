package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyConstitutionAmendment(t *testing.T) {
	govKeeper, _, _, ctx := setupGovKeeper(t)

	tests := []struct {
		name                string
		initialConstitution string
		amendment           string
		expectedResult      string
		expectError         bool
	}{
		{
			name:                "failed patch application",
			initialConstitution: "Hello World",
			amendment:           "Hi World",
			expectError:         true,
		},
		{
			name:                "successful patch application",
			initialConstitution: "Hello\nWorld",
			amendment:           "@@ -1,2 +1,2 @@\n-Hello\n+Hi\n World",
			expectError:         false,
			expectedResult:      "Hi\nWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			govKeeper.SetConstitution(ctx, tt.initialConstitution)
			updatedConstitution, err := govKeeper.ApplyConstitutionAmendment(ctx, tt.amendment)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, updatedConstitution)
			}
		})
	}
}
