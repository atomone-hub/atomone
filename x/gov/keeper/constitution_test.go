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
