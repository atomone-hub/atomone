package client

import (
	govclient "github.com/atomone-hub/atomone/x/gov/client"
	"github.com/atomone-hub/atomone/x/params/client/cli"
)

// ProposalHandler is the param change proposal handler.
var ProposalHandler = govclient.NewProposalHandler(cli.NewSubmitParamChangeProposalTxCmd)
