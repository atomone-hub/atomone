package ante_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/atomone-hub/atomone/ante"
	"github.com/atomone-hub/atomone/app/helpers"
	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// TestGovSubmitProposalDecoratorAtomOneV1 checks that the decorator rejects atomone
// gov v1 proposals that bundle a coredaos MsgUpdateParams changing the oversight
// DAO address with other messages.
func TestGovSubmitProposalDecoratorAtomOneV1(t *testing.T) {
	atomoneApp := helpers.Setup(t)
	ctx := atomoneApp.NewUncachedContext(true, tmproto.Header{})
	decorator := ante.NewGovSubmitProposalDecorator(atomoneApp.AppCodec(), atomoneApp.CoreDaosKeeper)

	addrs := simtestutil.CreateRandomAccounts(3)
	currentOversightAddr := addrs[0].String()
	newOversightAddr := addrs[1].String()
	proposer := addrs[2].String()
	extDuration := 7 * 24 * time.Hour

	// Set initial coredaos params with a known oversight DAO address.
	err := atomoneApp.CoreDaosKeeper.Params.Set(ctx, coredaostypes.Params{
		OversightDaoAddress:           currentOversightAddr,
		VotingPeriodExtensionDuration: &extDuration,
	})
	require.NoError(t, err)

	updateParamsChanging := &coredaostypes.MsgUpdateParams{
		Authority: proposer,
		Params: coredaostypes.Params{
			OversightDaoAddress:           newOversightAddr, // different from current
			VotingPeriodExtensionDuration: &extDuration,
		},
	}
	updateParamsSame := &coredaostypes.MsgUpdateParams{
		Authority: proposer,
		Params: coredaostypes.Params{
			OversightDaoAddress:           currentOversightAddr, // same as current
			VotingPeriodExtensionDuration: &extDuration,
		},
	}
	updateParamsSameUppercase := &coredaostypes.MsgUpdateParams{
		Authority: proposer,
		Params: coredaostypes.Params{
			OversightDaoAddress:           strings.ToUpper(currentOversightAddr), // same address, uppercased
			VotingPeriodExtensionDuration: &extDuration,
		},
	}
	otherMsg := govv1.NewMsgVote(addrs[2], 1, govv1.VoteOption_VOTE_OPTION_YES, "")

	tests := []struct {
		name       string
		msgs       []sdk.Msg
		expectPass bool
	}{
		{
			name:       "single MsgUpdateParams changing oversight DAO — allowed",
			msgs:       []sdk.Msg{mustNewAtomOneSubmitProposal(t, []sdk.Msg{updateParamsChanging}, proposer)},
			expectPass: true,
		},
		{
			name:       "single MsgUpdateParams not changing oversight DAO — allowed",
			msgs:       []sdk.Msg{mustNewAtomOneSubmitProposal(t, []sdk.Msg{updateParamsSame}, proposer)},
			expectPass: true,
		},
		{
			name:       "MsgUpdateParams (oversight change) bundled with other msg — rejected",
			msgs:       []sdk.Msg{mustNewAtomOneSubmitProposal(t, []sdk.Msg{updateParamsChanging, otherMsg}, proposer)},
			expectPass: false,
		},
		{
			name:       "MsgUpdateParams (no oversight change) bundled with other msg — allowed",
			msgs:       []sdk.Msg{mustNewAtomOneSubmitProposal(t, []sdk.Msg{updateParamsSame, otherMsg}, proposer)},
			expectPass: true,
		},
		{
			name:       "MsgUpdateParams (same address uppercased) bundled with other msg — allowed",
			msgs:       []sdk.Msg{mustNewAtomOneSubmitProposal(t, []sdk.Msg{updateParamsSameUppercase, otherMsg}, proposer)},
			expectPass: true,
		},
		{
			name:       "multiple non-MsgUpdateParams msgs — allowed",
			msgs:       []sdk.Msg{mustNewAtomOneSubmitProposal(t, []sdk.Msg{otherMsg, otherMsg}, proposer)},
			expectPass: true,
		},
		{
			name:       "non-submit-proposal message — allowed",
			msgs:       []sdk.Msg{otherMsg},
			expectPass: true,
		},
	}

	for _, tc := range tests {
		err := decorator.ValidateSubmitProposalMsgs(ctx, tc.msgs)
		if tc.expectPass {
			require.NoError(t, err, "expected %v to pass", tc.name)
		} else {
			require.Error(t, err, "expected %v to fail", tc.name)
		}
	}
}

// TestGovSubmitProposalDecoratorSDKV1 checks that the decorator rejects cosmos SDK
// gov v1 proposals that bundle a coredaos MsgUpdateParams changing the oversight
// DAO address with other messages.
func TestGovSubmitProposalDecoratorSDKV1(t *testing.T) {
	atomoneApp := helpers.Setup(t)
	ctx := atomoneApp.NewUncachedContext(true, tmproto.Header{})
	decorator := ante.NewGovSubmitProposalDecorator(atomoneApp.AppCodec(), atomoneApp.CoreDaosKeeper)

	addrs := simtestutil.CreateRandomAccounts(3)
	currentOversightAddr := addrs[0].String()
	newOversightAddr := addrs[1].String()
	proposer := addrs[2].String()
	extDuration := 7 * 24 * time.Hour

	// Set initial coredaos params with a known oversight DAO address.
	err := atomoneApp.CoreDaosKeeper.Params.Set(ctx, coredaostypes.Params{
		OversightDaoAddress:           currentOversightAddr,
		VotingPeriodExtensionDuration: &extDuration,
	})
	require.NoError(t, err)

	updateParamsChanging := &coredaostypes.MsgUpdateParams{
		Authority: proposer,
		Params: coredaostypes.Params{
			OversightDaoAddress:           newOversightAddr, // different from current
			VotingPeriodExtensionDuration: &extDuration,
		},
	}
	updateParamsSame := &coredaostypes.MsgUpdateParams{
		Authority: proposer,
		Params: coredaostypes.Params{
			OversightDaoAddress:           currentOversightAddr, // same as current
			VotingPeriodExtensionDuration: &extDuration,
		},
	}
	updateParamsSameUppercase := &coredaostypes.MsgUpdateParams{
		Authority: proposer,
		Params: coredaostypes.Params{
			OversightDaoAddress:           strings.ToUpper(currentOversightAddr), // same address, uppercased
			VotingPeriodExtensionDuration: &extDuration,
		},
	}
	otherMsg := govv1.NewMsgVote(addrs[2], 1, govv1.VoteOption_VOTE_OPTION_YES, "")

	tests := []struct {
		name       string
		msgs       []sdk.Msg
		expectPass bool
	}{
		{
			name:       "single MsgUpdateParams changing oversight DAO — allowed",
			msgs:       []sdk.Msg{mustNewSDKSubmitProposal(t, []sdk.Msg{updateParamsChanging}, proposer)},
			expectPass: true,
		},
		{
			name:       "single MsgUpdateParams not changing oversight DAO — allowed",
			msgs:       []sdk.Msg{mustNewSDKSubmitProposal(t, []sdk.Msg{updateParamsSame}, proposer)},
			expectPass: true,
		},
		{
			name:       "MsgUpdateParams (oversight change) bundled with other msg — rejected",
			msgs:       []sdk.Msg{mustNewSDKSubmitProposal(t, []sdk.Msg{updateParamsChanging, otherMsg}, proposer)},
			expectPass: false,
		},
		{
			name:       "MsgUpdateParams (no oversight change) bundled with other msg — allowed",
			msgs:       []sdk.Msg{mustNewSDKSubmitProposal(t, []sdk.Msg{updateParamsSame, otherMsg}, proposer)},
			expectPass: true,
		},
		{
			name:       "MsgUpdateParams (same address uppercased) bundled with other msg — allowed",
			msgs:       []sdk.Msg{mustNewSDKSubmitProposal(t, []sdk.Msg{updateParamsSameUppercase, otherMsg}, proposer)},
			expectPass: true,
		},
		{
			name:       "multiple non-MsgUpdateParams msgs — allowed",
			msgs:       []sdk.Msg{mustNewSDKSubmitProposal(t, []sdk.Msg{otherMsg, otherMsg}, proposer)},
			expectPass: true,
		},
		{
			name:       "non-submit-proposal message — allowed",
			msgs:       []sdk.Msg{otherMsg},
			expectPass: true,
		},
	}

	for _, tc := range tests {
		err := decorator.ValidateSubmitProposalMsgs(ctx, tc.msgs)
		if tc.expectPass {
			require.NoError(t, err, "expected %v to pass", tc.name)
		} else {
			require.Error(t, err, "expected %v to fail", tc.name)
		}
	}
}

func mustNewAtomOneSubmitProposal(t *testing.T, msgs []sdk.Msg, proposer string) *govv1.MsgSubmitProposal {
	t.Helper()
	msg, err := govv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(), proposer, "", "title", "summary")
	require.NoError(t, err)
	return msg
}

func mustNewSDKSubmitProposal(t *testing.T, msgs []sdk.Msg, proposer string) *sdkgovv1.MsgSubmitProposal {
	t.Helper()
	msg, err := sdkgovv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(), proposer, "", "title", "summary")
	require.NoError(t, err)
	return msg
}
