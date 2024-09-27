package atomone

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	paramscutils "github.com/cosmos/cosmos-sdk/x/params/client/utils"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/upgrade/plan"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	govclient "github.com/atomone-hub/atomone/x/gov/client"
	govcli "github.com/atomone-hub/atomone/x/gov/client/cli"
	govv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

var (
	paramsChangeProposalHandler    = govclient.NewProposalHandler(newSubmitParamChangeProposalTxCmd)
	upgradeProposalHandler         = govclient.NewProposalHandler(newCmdSubmitLegacyUpgradeProposal)
	cancelUpgradeProposalHandler   = govclient.NewProposalHandler(newCmdSubmitLegacyCancelUpgradeProposal)
	updateIBCClientProposalHandler = govclient.NewProposalHandler(newCmdSubmitUpdateIBCClientProposal)
	upgradeIBCProposalHandler      = govclient.NewProposalHandler(NewCmdSubmitUpgradeProposal)
)

func init() {
	// Proposal types are registered within their specific module in the SDK, but
	// using the legacy gov module. To register them in the atomone gov module,
	// we need to do it here.
	govv1beta1.RegisterProposalType(paramproposal.ProposalTypeChange)
	govv1beta1.RegisterProposalType(upgradetypes.ProposalTypeSoftwareUpgrade)
	govv1beta1.RegisterProposalType(upgradetypes.ProposalTypeCancelSoftwareUpgrade)
}

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

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
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

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal") //nolint:staticcheck // we are intentionally using a deprecated flag here.
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
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

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription) //nolint:staticcheck // we are intentionally using a deprecated flag here.
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

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal") //nolint:staticcheck // we are intentionally using a deprecated flag here.
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.MarkFlagRequired(govcli.FlagTitle)       //nolint:errcheck
	cmd.MarkFlagRequired(govcli.FlagDescription) //nolint:staticcheck,errcheck // we are intentionally using a deprecated flag here.

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
	title, err := fs.GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := fs.GetString(govcli.FlagDescription) //nolint:staticcheck // we are intentionally using a deprecated flag here.
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

// newCmdSubmitUpdateIBCClientProposal implements a command handler for submitting an update IBC client proposal transaction.
func newCmdSubmitUpdateIBCClientProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-client [subject-client-id] [substitute-client-id]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit an update IBC client proposal",
		Long: "Submit an update IBC client proposal along with an initial deposit.\n" +
			"Please specify a subject client identifier you want to update..\n" +
			"Please specify the substitute client the subject client will be updated to.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(govcli.FlagTitle) //nolint:staticcheck // need this till full govv1 conversion.
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription) //nolint:staticcheck // need this till full govv1 conversion.
			if err != nil {
				return err
			}

			subjectClientID := args[0]
			substituteClientID := args[1]

			content := ibcclienttypes.NewClientUpdateProposal(title, description, subjectClientID, substituteClientID)

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
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

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")             //nolint:staticcheck // need this till full govv1 conversion.
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal") //nolint:staticcheck // need this till full govv1 conversion.
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	return cmd
}

// NewCmdSubmitUpgradeProposal implements a command handler for submitting an upgrade IBC client proposal transaction.
func NewCmdSubmitUpgradeProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ibc-upgrade [name] [height] [path/to/upgraded_client_state.json] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Submit an IBC upgrade proposal",
		Long: "Submit an IBC client breaking upgrade proposal along with an initial deposit.\n" +
			"The client state specified is the upgraded client state representing the upgraded chain\n" +
			`Example Upgraded Client State JSON: 
{
	"@type":"/ibc.lightclients.tendermint.v1.ClientState",
 	"chain_id":"testchain1",
	"unbonding_period":"1814400s",
	"latest_height":{"revision_number":"0","revision_height":"2"},
	"proof_specs":[{"leaf_spec":{"hash":"SHA256","prehash_key":"NO_HASH","prehash_value":"SHA256","length":"VAR_PROTO","prefix":"AA=="},"inner_spec":{"child_order":[0,1],"child_size":33,"min_prefix_length":4,"max_prefix_length":12,"empty_child":null,"hash":"SHA256"},"max_depth":0,"min_depth":0},{"leaf_spec":{"hash":"SHA256","prehash_key":"NO_HASH","prehash_value":"SHA256","length":"VAR_PROTO","prefix":"AA=="},"inner_spec":{"child_order":[0,1],"child_size":32,"min_prefix_length":1,"max_prefix_length":1,"empty_child":null,"hash":"SHA256"},"max_depth":0,"min_depth":0}],
	"upgrade_path":["upgrade","upgradedIBCState"],
}
			`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

			title, err := cmd.Flags().GetString(govcli.FlagTitle) //nolint:staticcheck // need this till full govv1 conversion.
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription) //nolint:staticcheck // need this till full govv1 conversion.
			if err != nil {
				return err
			}

			name := args[0]

			height, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}

			plan := upgradetypes.Plan{
				Name:   name,
				Height: height,
			}

			// attempt to unmarshal client state argument
			var clientState ibcexported.ClientState
			clientContentOrFileName := args[2]
			if err := cdc.UnmarshalInterfaceJSON([]byte(clientContentOrFileName), &clientState); err != nil {

				// check for file path if JSON input is not provided
				contents, err := os.ReadFile(clientContentOrFileName)
				if err != nil {
					return fmt.Errorf("neither JSON input nor path to .json file for client state were provided: %w", err)
				}

				if err := cdc.UnmarshalInterfaceJSON(contents, &clientState); err != nil {
					return fmt.Errorf("error unmarshalling client state file: %w", err)
				}
			}

			content, err := ibcclienttypes.NewUpgradeProposal(title, description, plan, clientState)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
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

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")             //nolint:staticcheck // need this till full govv1 conversion.
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal") //nolint:staticcheck // need this till full govv1 conversion.
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	return cmd
}
