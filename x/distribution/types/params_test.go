package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/x/distribution/types"
)

func TestParams_ValidateBasic(t *testing.T) {
	toDec := sdkmath.LegacyMustNewDecFromStr
	tests := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{
			name: "success",
			params: types.Params{
				CommunityTax:             toDec("0.1"),
				NakamotoBonusCoefficient: toDec("0.3"),
				WithdrawAddrEnabled:      false,
			},
			wantErr: false,
		},
		{
			name: "negative community tax",
			params: types.Params{CommunityTax: toDec("-0.1"),
				NakamotoBonusCoefficient: toDec("0.3"),
				WithdrawAddrEnabled:      false,
			},
			wantErr: true,
		},
		{
			name: "negative nakamoto bonus coefficient",
			params: types.Params{
				CommunityTax:             toDec("0.1"),
				NakamotoBonusCoefficient: toDec("-0.3"),
				WithdrawAddrEnabled:      false,
			},
			wantErr: true,
		},
		{
			name: "nil nakamoto bonus coefficient",
			params: types.Params{
				CommunityTax:             toDec("0.1"),
				NakamotoBonusCoefficient: sdkmath.LegacyDec{},
				WithdrawAddrEnabled:      false,
			},
			wantErr: true,
		},
		{
			name: "nakamoto bonus coefficient greater than 1",
			params: types.Params{
				CommunityTax:             toDec("0.1"),
				NakamotoBonusCoefficient: toDec("1.1"),
				WithdrawAddrEnabled:      false,
			},
			wantErr: true,
		},
		{
			name: "total sum greater than 1 (must not matter)",
			params: types.Params{
				CommunityTax:             toDec("0.2"),
				NakamotoBonusCoefficient: toDec("0.3"),
				WithdrawAddrEnabled:      false,
			},
			wantErr: false,
		},
		{
			name: "community tax greater than 1",
			params: types.Params{
				CommunityTax:             toDec("1.1"),
				NakamotoBonusCoefficient: toDec("0.3"),
				WithdrawAddrEnabled:      false,
			},
			wantErr: true,
		},
		{
			name: "community tax nil",
			params: types.Params{
				CommunityTax:             sdkmath.LegacyDec{},
				NakamotoBonusCoefficient: toDec("0.3"),
				WithdrawAddrEnabled:      false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.params.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {
	require.NoError(t, types.DefaultParams().ValidateBasic())
}
