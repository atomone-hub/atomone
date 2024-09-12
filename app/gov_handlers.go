package atomone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/version"

	"github.com/atomone-hub/atomone/client"
	"github.com/atomone-hub/atomone/client/tx"
	sdk "github.com/atomone-hub/atomone/types"
	govclient "github.com/atomone-hub/atomone/x/gov/client"
	"github.com/atomone-hub/atomone/x/gov/client/cli"
	govv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	paramscutils "github.com/atomone-hub/atomone/x/params/client/utils"
	paramproposal "github.com/atomone-hub/atomone/x/params/types/proposal"
	"github.com/atomone-hub/atomone/x/upgrade/plan"
	upgradetypes "github.com/atomone-hub/atomone/x/upgrade/types"
)

var (
	paramsChangeProposalHandler  = govclient.NewProposalHandler(newSubmitParamChangeProposalTxCmd)
	upgradeProposalHandler       = govclient.NewProposalHandler(newCmdSubmitLegacyUpgradeProposal)
	cancelUpgradeProposalHandler = govclient.NewProposalHandler(newCmdSubmitLegacyCancelUpgradeProposal)
)

// NewSubmitParamChangeProposalTxCmd returns a CLI command handler for creating
// a parameter change proposal governance transaction.
//
// NOTE: copy of x/params/client.newSubmitParamChangeProposalTxCmd() except
// that it creates a atomone.gov.MsgSubmitProposal instead of a
// cosmos.gov.MsgSubmitProposal.
func newSubmitParamChangeProposalTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "param-change [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a parameter change proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a parameter proposal along with an initial deposit.
The proposal details must be supplied via a JSON file. For values that contains
objects, only non-empty fields will be updated.

IMPORTANT: Currently parameter changes are evaluated but not validated, so it is
very important that any "value" change is valid (ie. correct type and within bounds)
for its respective parameter, eg. "MaxValidators" should be an integer and not a decimal.

Proper vetting of a parameter change proposal should prevent this from happening
(no deposits should occur during the governance process), but it should be noted
regardless.

Example:
$ %s tx gov submit-proposal param-change <path/to/proposal.json> --from=<key_or_address>

Where proposal.json contains:

{
  "title": "Staking Param Change",
  "description": "Update max validators",
  "changes": [
    {
      "subspace": "staking",
      "key": "MaxValidators",
      "value": 105
    }
  ],
  "deposit": "1000stake"
}
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := paramscutils.ParseParamChangeProposalJSON(clientCtx.LegacyAmino, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := paramproposal.NewParameterChangeProposal(
				proposal.Title, proposal.Description, proposal.Changes.ToParamChanges(),
			)

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
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
