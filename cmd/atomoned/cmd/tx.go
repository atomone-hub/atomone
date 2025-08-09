package cmd

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/flags"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
)

func GetBroadCastCommand() *cobra.Command {
	cmd := authcmd.GetBroadcastCommand()

	// Hide the flags that are not needed/used for the broadcast command
	// but are added from the `flags.AddTxFlagsToCmd` function
	unusedFlags := []string{
		flags.FlagFrom,
		flags.FlagAccountNumber,
		flags.FlagSequence,
		flags.FlagNote,
		flags.FlagFees,
		flags.FlagGasPrices,
		flags.FlagUseLedger,
		flags.FlagGasAdjustment,
		flags.FlagDryRun,
		flags.FlagGenerateOnly,
		flags.FlagOffline,
		flags.FlagSkipConfirmation,
		flags.FlagSignMode,
		flags.FlagTimeoutHeight,
		flags.FlagFeePayer,
		flags.FlagFeeGranter,
		flags.FlagTip,
		flags.FlagAux,
		flags.FlagChainID,
		flags.FlagGas,
		flags.FlagKeyringDir,
		flags.FlagKeyringBackend,
	}

	for _, flag := range unusedFlags {
		_ = cmd.Flags().MarkHidden(flag)
	}

	return cmd
}
