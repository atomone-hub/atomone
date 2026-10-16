package v1_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// TestMsgUpdateParamsQuorumCheckDisabledNilParams is a regression test for the
// consensus-halt nil dereference: gov params with QuorumCheckCount == 0 used
// to skip quorum-check param validation, so a MsgUpdateParams could store
// MaxVotingPeriodExtension == nil (and QuorumTimeout == nil). The EndBlocker
// (cosmos-sdk-pin/x/gov/abci.go) dereferences *params.MaxVotingPeriodExtension
// for quorum check queue entries that were created while checks were enabled
// and persist across params updates, panicking in FinalizeBlock (no recover)
// on every validator.
func TestMsgUpdateParamsQuorumCheckDisabledNilParams(t *testing.T) {
	authority := sdk.AccAddress("authority_module_account").String()

	testCases := []struct {
		name     string
		mutate   func(*v1.Params)
		expError bool
	}{
		{
			name:     "valid: quorum checks disabled, params fully set",
			mutate:   func(p *v1.Params) { p.QuorumCheckCount = 0 },
			expError: false,
		},
		{
			name: "invalid: count 0 with nil MaxVotingPeriodExtension",
			mutate: func(p *v1.Params) {
				p.QuorumCheckCount = 0
				p.MaxVotingPeriodExtension = nil
			},
			expError: true,
		},
		{
			name: "invalid: count 0 with nil QuorumTimeout",
			mutate: func(p *v1.Params) {
				p.QuorumCheckCount = 0
				p.QuorumTimeout = nil
			},
			expError: true,
		},
		{
			name: "invalid: count enabled with nil MaxVotingPeriodExtension",
			mutate: func(p *v1.Params) {
				p.QuorumCheckCount = 1
				p.MaxVotingPeriodExtension = nil
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := v1.DefaultParams()
			tc.mutate(&params)
			msg := v1.MsgUpdateParams{
				Authority: authority,
				Params:    params,
			}
			err := msg.ValidateBasic()
			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSDKParamsQuorumCheckDisabledNilParams checks the fork-side gate: the
// pinned Cosmos SDK Params.ValidateBasic is the last validation before
// Params.Set on the UpdateParams store path, so it must reject the same
// shape.
func TestSDKParamsQuorumCheckDisabledNilParams(t *testing.T) {
	params := sdkgovv1.DefaultParams()
	params.QuorumCheckCount = 0
	params.MaxVotingPeriodExtension = nil
	require.Error(t, params.ValidateBasic(), "nil MaxVotingPeriodExtension must be rejected even when quorum checks are disabled")

	params = sdkgovv1.DefaultParams()
	params.QuorumCheckCount = 0
	params.QuorumTimeout = nil
	require.Error(t, params.ValidateBasic(), "nil QuorumTimeout must be rejected even when quorum checks are disabled")

	// Default params (quorum checks disabled by default, non-nil values) must
	// remain valid.
	require.NoError(t, sdkgovv1.DefaultParams().ValidateBasic())
	require.NoError(t, v1.DefaultParams().ValidateBasic())
}
