package snapshot

import (
	"github.com/spf13/cobra"

	servertypes "github.com/atomone-hub/atomone/server/types"
)

// Cmd returns the snapshots group command
func Cmd(appCreator servertypes.AppCreator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "Manage local snapshots",
	}
	cmd.AddCommand(
		ListSnapshotsCmd,
		RestoreSnapshotCmd(appCreator),
		ExportSnapshotCmd(appCreator),
		DumpArchiveCmd(),
		LoadArchiveCmd(),
		DeleteSnapshotCmd(),
	)
	return cmd
}
