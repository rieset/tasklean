package commands

import "github.com/spf13/cobra"

func newRemoteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage remote trackers",
		Long:  "Add, remove, and list remote task trackers",
	}

	cmd.AddCommand(
		newRemoteAddCommand(),
		newRemoteRemoveCommand(),
		newRemoteConfigCommand(),
	)

	return cmd
}
