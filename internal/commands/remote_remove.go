package commands

import (
	"fmt"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/tui"
	"github.com/spf13/cobra"
)

func newRemoteRemoveCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a remote tracker",
		Long:  "Remove a remote tracker configuration. Asks for confirmation (Bubble Tea TUI). Use --force to skip confirmation.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if !config.RemoteConfigExists(name) {
				return fmt.Errorf("remote %q does not exist", name)
			}

			if !force {
				confirmed, err := tui.Confirm(fmt.Sprintf("Remove remote %q?", name))
				if err != nil {
					return fmt.Errorf("confirmation cancelled: %w", err)
				}
				if !confirmed {
					fmt.Println("Cancelled")
					return nil
				}
			}

			if err := config.RemoveRemoteConfig(name); err != nil {
				return fmt.Errorf("failed to remove remote config: %w", err)
			}

			fmt.Printf("Removed remote %q\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
