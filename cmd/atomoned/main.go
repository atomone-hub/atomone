package main

import (
	"os"

	"github.com/atomone-hub/atomone/server"
	svrcmd "github.com/atomone-hub/atomone/server/cmd"

	app "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/cmd/atomoned/cmd"
)

func main() {
	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
