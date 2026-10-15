package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	app "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/cmd/atomoned/cmd"
	"github.com/atomone-hub/atomone/pkg/ledger"
)

func main() {
	// Ledger devices are reached without CGO, so in every build.
	ledger.Register()

	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
