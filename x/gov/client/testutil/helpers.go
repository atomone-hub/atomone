package testutil

import (
	"fmt"
	"time"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"

	govcli "github.com/atomone-hub/atomone/x/gov/client/cli"
	"github.com/atomone-hub/atomone/x/gov/keeper"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(10))).String()),
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

func HasActiveProposal(ctx sdk.Context, k *keeper.Keeper, id uint64, t time.Time) bool {
	it := k.ActiveProposalQueueIterator(ctx, t)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		proposalID, _ := types.SplitActiveProposalQueueKey(it.Key())
		if proposalID == id {
			return true
		}
	}
	return false
}

func HasInactiveProposal(ctx sdk.Context, k *keeper.Keeper, id uint64, t time.Time) bool {
	it := k.InactiveProposalQueueIterator(ctx, t)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		proposalID, _ := types.SplitInactiveProposalQueueKey(it.Key())
		if proposalID == id {
			return true
		}
	}
	return false
}

func HasQuorumCheck(ctx sdk.Context, k *keeper.Keeper, id uint64, t time.Time) bool {
	_, ok := GetQuorumCheckQueueEntry(ctx, k, id, t)
	return ok
}

func GetQuorumCheckQueueEntry(ctx sdk.Context, k *keeper.Keeper, id uint64, t time.Time) (v1.QuorumCheckQueueEntry, bool) {
	it := k.QuorumCheckQueueIterator(ctx, t)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		proposalID, _ := types.SplitQuorumQueueKey(it.Key())
		if proposalID == id {
			bz := it.Value()
			var q v1.QuorumCheckQueueEntry
			err := q.Unmarshal(bz)
			return q, err == nil
		}
	}
	return v1.QuorumCheckQueueEntry{}, false
}
