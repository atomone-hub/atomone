package authz

import (
	"github.com/atomone-hub/atomone/client"
	"github.com/atomone-hub/atomone/testutil"
	clitestutil "github.com/atomone-hub/atomone/testutil/cli"
	"github.com/atomone-hub/atomone/x/authz/client/cli"
)

func CreateGrant(clientCtx client.Context, args []string) (testutil.BufferWriter, error) {
	cmd := cli.NewCmdGrantAuthorization()
	return clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
}
