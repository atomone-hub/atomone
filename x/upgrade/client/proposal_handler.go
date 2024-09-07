package client

import (
	govclient "github.com/atomone-hub/atomone/x/gov/client"
	"github.com/atomone-hub/atomone/x/upgrade/client/cli"
)

var (
	LegacyProposalHandler       = govclient.NewProposalHandler(cli.NewCmdSubmitLegacyUpgradeProposal)
	LegacyCancelProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitLegacyCancelUpgradeProposal)
)
