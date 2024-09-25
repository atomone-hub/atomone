package atomone

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade/plan"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	govclient "github.com/atomone-hub/atomone/x/gov/client"
	"github.com/atomone-hub/atomone/x/gov/client/cli"
	govv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

var (
	upgradeProposalHandler       = govclient.NewProposalHandler(newCmdSubmitLegacyUpgradeProposal)
	cancelUpgradeProposalHandler = govclient.NewProposalHandler(newCmdSubmitLegacyCancelUpgradeProposal)
)

func init() {
	// Proposal types are registered within their specific module in the SDK, but
	// using the legacy gov module. To register them in the atomone gov module,
	// we need to do it here.
	govv1beta1.RegisterProposalType(upgradetypes.ProposalTypeSoftwareUpgrade)
	govv1beta1.RegisterProposalType(upgradetypes.ProposalTypeCancelSoftwareUpgrade)
}

const (
	// Deprecated: only used for v1beta1 legacy proposals.
	FlagUpgradeHeight = "upgrade-height"
	// Deprecated: only used for v1beta1 legacy proposals.
	FlagUpgradeInfo = "upgrade-info"
	FlagNoValidate  = "no-validate"
	FlagDaemonName  = "daemon-name"
)

// newCmdSubmitLegacyUpgradeProposal implements a command handler for submitting a software upgrade proposal transaction.
// Deprecated: please use NewCmdSubmitUpgradeProposal instead.ck
//
// NOTE: copy of x/upgrade/client.NewCmdSubmitUpgradeProposal() except
// that it creates a atomone.gov.MsgSubmitProposal instead of a
// cosmos.gov.MsgSubmitProposal.
func newCmdSubmitLegacyUpgradeProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "software-upgrade [name] (--upgrade-height [height]) (--upgrade-info [info]) [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a software upgrade proposal",
		Long: "Submit a software upgrade along with an initial deposit.\n" +
			"Please specify a unique name and height for the upgrade to take effect.\n" +
			"You may include info to reference a binary download link, in a format compatible with: https://github.com/cosmos/cosmos-sdk/tree/main/cosmovisor",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			name := args[0]
			content, err := parseArgsToContent(cmd.Flags(), name)
			if err != nil {
				return err
			}
			noValidate, err := cmd.Flags().GetBool(FlagNoValidate)
			if err != nil {
				return err
			}
			if !noValidate {
				prop := content.(*upgradetypes.SoftwareUpgradeProposal) //nolint:staticcheck // we are intentionally using a deprecated proposal type.
				var daemonName string
				if daemonName, err = cmd.Flags().GetString(FlagDaemonName); err != nil {
					return err
				}
				var planInfo *plan.Info
				if planInfo, err = plan.ParseInfo(prop.Plan.Info); err != nil {
					return err
				}
				if err = planInfo.ValidateFull(daemonName); err != nil {
					return err
				}
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal") //nolint:staticcheck // we are intentionally using a deprecated flag here.
	cmd.Flags().String(cli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().Int64(FlagUpgradeHeight, 0, "The height at which the upgrade must happen")
	cmd.Flags().String(FlagUpgradeInfo, "", "Info for the upgrade plan such as new version download urls, etc.")
	cmd.Flags().Bool(FlagNoValidate, false, "Skip validation of the upgrade info")
	cmd.Flags().String(FlagDaemonName, getDefaultDaemonName(), "The name of the executable being upgraded (for upgrade-info validation). Default is the DAEMON_NAME env var if set, or else this executable")

	return cmd
}

// newCmdSubmitLegacyCancelUpgradeProposal implements a command handler for submitting a software upgrade cancel proposal transaction.
// Deprecated: please use NewCmdSubmitCancelUpgradeProposal instead.
//
// NOTE: copy of x/upgrade/client.newcmdsubmitcancelupgradeproposal() except
// that it creates a atomone.gov.msgsubmitproposal instead of a
// cosmos.gov.msgsubmitproposal.
func newCmdSubmitLegacyCancelUpgradeProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-software-upgrade [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Cancel the current software upgrade proposal",
		Long:  "Cancel a software upgrade along with an initial deposit.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription) //nolint:staticcheck // we are intentionally using a deprecated flag here.
			if err != nil {
				return err
			}

			content := upgradetypes.NewCancelSoftwareUpgradeProposal(title, description)

			msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal") //nolint:staticcheck // we are intentionally using a deprecated flag here.
	cmd.Flags().String(cli.FlagDeposit, "", "deposit of proposal")
	cmd.MarkFlagRequired(cli.FlagTitle)       //nolint:errcheck
	cmd.MarkFlagRequired(cli.FlagDescription) //nolint:staticcheck,errcheck // we are intentionally using a deprecated flag here.

	return cmd
}

// getDefaultDaemonName gets the default name to use for the daemon.
// If a DAEMON_NAME env var is set, that is used.
// Otherwise, the last part of the currently running executable is used.
func getDefaultDaemonName() string {
	// DAEMON_NAME is specifically used here to correspond with the Cosmovisor setup env vars.
	name := os.Getenv("DAEMON_NAME")
	if len(name) == 0 {
		_, name = filepath.Split(os.Args[0])
	}
	return name
}

func parseArgsToContent(fs *pflag.FlagSet, name string) (govv1beta1.Content, error) {
	title, err := fs.GetString(cli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := fs.GetString(cli.FlagDescription) //nolint:staticcheck // we are intentionally using a deprecated flag here.
	if err != nil {
		return nil, err
	}

	height, err := fs.GetInt64(FlagUpgradeHeight)
	if err != nil {
		return nil, err
	}

	info, err := fs.GetString(FlagUpgradeInfo)
	if err != nil {
		return nil, err
	}

	plan := upgradetypes.Plan{Name: name, Height: height, Info: info}
	content := upgradetypes.NewSoftwareUpgradeProposal(title, description, plan)
	return content, nil
}
