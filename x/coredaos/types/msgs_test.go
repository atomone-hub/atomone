package types_test

import (
	"github.com/atomone-hub/atomone/x/coredaos/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	addrs = []sdk.AccAddress{
		sdk.AccAddress("test1"),
		sdk.AccAddress("test2"),
	}
)

func TestMsgAnnotateProposal_ValidateBasic(t *testing.T) {
	tests := []struct {
		annotator  sdk.AccAddress
		proposalId uint64
		annotation string
		expectPass bool
	}{
		{sdk.AccAddress{}, 0, "annotation", false},
		{addrs[0], 0, "", false},
		{addrs[0], 0, "annotation", true},
	}
	for i, tc := range tests {
		msg := types.NewMsgAnnotateProposal(tc.annotator, tc.proposalId, tc.annotation)
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgExtendVotingPeriod_ValidateBasic(t *testing.T) {
	tests := []struct {
		extender   sdk.AccAddress
		proposalId uint64
		expectPass bool
	}{
		{sdk.AccAddress{}, 0, false},
		{addrs[0], 0, true},
	}
	for i, tc := range tests {
		msg := types.NewMsgExtendVotingPeriod(tc.extender, tc.proposalId)
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgEndorseProposal_ValidateBasic(t *testing.T) {
	tests := []struct {
		endorser   sdk.AccAddress
		proposalId uint64
		expectPass bool
	}{
		{sdk.AccAddress{}, 0, false},
		{addrs[0], 0, true},
	}
	for i, tc := range tests {
		msg := types.NewMsgEndorseProposal(tc.endorser, tc.proposalId)
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgVetoProposal_ValidateBasic(t *testing.T) {
	tests := []struct {
		vetoer      sdk.AccAddress
		proposalId  uint64
		burnDeposit bool
		expectPass  bool
	}{
		{sdk.AccAddress{}, 0, true, false},
		{sdk.AccAddress{}, 0, false, false},
		{addrs[0], 0, true, true},
		{addrs[0], 0, false, true},
	}
	for i, tc := range tests {
		msg := types.NewMsgEndorseProposal(tc.vetoer, tc.proposalId)
		if tc.expectPass {
			require.NoError(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.Error(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}
