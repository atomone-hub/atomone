package main

import (
	"os"

	"github.com/atomone-hub/atomone/server"
	svrcmd "github.com/atomone-hub/atomone/server/cmd"
	"github.com/atomone-hub/atomone/simapp"
	"github.com/atomone-hub/atomone/simapp/simd/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", simapp.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
