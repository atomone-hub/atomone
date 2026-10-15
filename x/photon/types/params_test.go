package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/x/photon/types"
)

func TestParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{
			name:   "default params",
			params: types.DefaultParams(),
		},
		{
			name:   "empty exceptions",
			params: types.NewParams(false, nil),
		},
		{
			name:   "single specific exception",
			params: types.NewParams(false, []string{"/atomone.photon.v1.MsgMintPhoton"}),
		},
		{
			name: "multiple specific exceptions",
			params: types.NewParams(false, []string{
				"/atomone.photon.v1.MsgMintPhoton",
				"/cosmos.bank.v1beta1.MsgSend",
			}),
		},
		{
			name:   "wildcard alone",
			params: types.NewParams(false, []string{"*"}),
		},
		{
			name:    "wildcard at index 0 mixed with specific",
			params:  types.NewParams(false, []string{"*", "/atomone.photon.v1.MsgMintPhoton"}),
			wantErr: true,
		},
		{
			name:    "wildcard at index 1 mixed with specific",
			params:  types.NewParams(false, []string{"/atomone.photon.v1.MsgMintPhoton", "*"}),
			wantErr: true,
		},
		{
			name: "wildcard at index 2 mixed with specific",
			params: types.NewParams(false, []string{
				"/atomone.photon.v1.MsgMintPhoton",
				"/cosmos.gov.v1.MsgVote",
				"*",
			}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
