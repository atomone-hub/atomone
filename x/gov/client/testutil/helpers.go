package testutil

import (
	"fmt"

	"github.com/atomone-hub/atomone/client"
	"github.com/atomone-hub/atomone/client/flags"
	"github.com/atomone-hub/atomone/testutil"
	clitestutil "github.com/atomone-hub/atomone/testutil/cli"
	sdk "github.com/atomone-hub/atomone/types"
	govcli "github.com/atomone-hub/atomone/x/gov/client/cli"
)

var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
}

// MsgSubmitLegacyProposal creates a tx for submit legacy proposal
//
//nolint:staticcheck // we are intentionally using a deprecated flag here.
func MsgSubmitLegacyProposal(clientCtx client.Context, from, title, description, proposalType string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := append([]string{
		fmt.Sprintf("--%s=%s", govcli.FlagTitle, title),
		fmt.Sprintf("--%s=%s", govcli.FlagDescription, description),
		fmt.Sprintf("--%s=%s", govcli.FlagProposalType, proposalType),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from),
	}, commonArgs...)

	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, govcli.NewCmdSubmitLegacyProposal(), args)
}

// MsgVote votes for a proposal
func MsgVote(clientCtx client.Context, from, id, vote string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := append([]string{
		id,
		vote,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from),
	}, commonArgs...)

	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, govcli.NewCmdWeightedVote(), args)
}

// MsgDeposit deposits on a proposal
func MsgDeposit(clientCtx client.Context, from, id, deposit string, extraArgs ...string) (testutil.BufferWriter, error) {
	args := append([]string{
		id,
		deposit,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from),
	}, commonArgs...)

	args = append(args, extraArgs...)

	return clitestutil.ExecTestCLICmd(clientCtx, govcli.NewCmdDeposit(), args)
}