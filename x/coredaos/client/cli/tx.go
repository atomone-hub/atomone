package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/atomone-hub/atomone/x/coredaos/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		GetTxAnnotateProposalCmd(),
		GetTxEndorseProposalCmd(),
		GetTxExtendVotingPeriodCmd(),
		GetTxVetoProposalCmd(),
	)
	return cmd
}

// GetTxAnnotateProposalCmd returns the command to annotate a proposal
func GetTxAnnotateProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotate [proposal-id] [annotation]",
		Short: "Broadcast a message to annotate a proposal. Only available to the Steering DAO.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("proposal-id %s not a valid uint, please input a valid proposal-id", args[0])
			}
			msg := types.NewMsgAnnotateProposal(
				clientCtx.GetFromAddress(),
				proposalID,
				args[1],
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetTxEndorseProposalCmd returns the command to endorse a proposal
func GetTxEndorseProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorse [proposal-id]",
		Short: "Broadcast a message to endorse a proposal. Only available to the Steering DAO.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("proposal-id %s not a valid uint, please input a valid proposal-id", args[0])
			}
			msg := types.NewMsgEndorseProposal(
				clientCtx.GetFromAddress(),
				proposalID,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetTxExtendVotingPeriodCmd returns the command to extend the voting period of a proposal
func GetTxExtendVotingPeriodCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extend-voting-period [proposal-id]",
		Short: "Broadcast a message to extend the voting period of a proposal. Only available to the Steering DAO and Oversight DAO.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("proposal-id %s not a valid uint, please input a valid proposal-id", args[0])
			}
			msg := types.NewMsgExtendVotingPeriod(
				clientCtx.GetFromAddress(),
				proposalID,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetTxVetoProposalCmd returns the command to veto a proposal
func GetTxVetoProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "veto [proposal-id] [burn-deposit]",
		Short: "Broadcast a message to veto a proposal. Only available to the Oversight DAO.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("proposal-id %s not a valid uint, please input a valid proposal-id", args[0])
			}
			burnDeposit, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("burn-deposit %s not a valid boolean, please input a valid burn-deposit", args[1])
			}
			msg := types.NewMsgVetoProposal(
				clientCtx.GetFromAddress(),
				proposalID,
				burnDeposit,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
