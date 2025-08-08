package types_test

import (
	"testing"
	"time"

	"github.com/atomone-hub/atomone/x/coredaos/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState func() *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis,
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: func() *types.GenesisState {
				params := types.DefaultParams()
				return &types.GenesisState{Params: params}
			},
			valid: true,
		},
		{
			desc: "invalid genesis state nil extension duration",
			genState: func() *types.GenesisState {
				params := types.DefaultParams()
				params.VotingPeriodExtensionDuration = nil
				return &types.GenesisState{Params: params}
			},
			valid: false,
		},
		{
			desc: "invalid genesis state wrong oversightdaoaddress",
			genState: func() *types.GenesisState {
				params := types.DefaultParams()
				params.OversightDaoAddress = "cosmosincorrectaddress"
				return &types.GenesisState{Params: params}
			},
			valid: false,
		},
		{
			desc: "invalid genesis state wrong steeringdaoaddress",
			genState: func() *types.GenesisState {
				params := types.DefaultParams()
				params.SteeringDaoAddress = "cosmosincorrectaddress"
				return &types.GenesisState{Params: params}
			},
			valid: false,
		},
		{
			desc: "invalid genesis state negative extension duration",
			genState: func() *types.GenesisState {
				negativeTimeDuration := time.Duration(-1)
				params := types.DefaultParams()
				params.VotingPeriodExtensionDuration = &negativeTimeDuration
				return &types.GenesisState{Params: params}
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState().Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
