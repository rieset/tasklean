package commands

import (
	"fmt"
	"os"

	"github.com/rie/tasklean/internal/config"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show sync status",
		RunE: func(cmd *cobra.Command, args []string) error {
			remotes, err := config.ListRemoteConfigs()
			if err != nil {
				return fmt.Errorf("failed to list remotes: %w", err)
			}

			if len(remotes) == 0 {
				fmt.Println("No remotes configured")
				fmt.Println("Use 'tasklean remote add <name> <url>' to add a remote")
				return nil
			}

			fmt.Printf("Configured remotes (%d):\n\n", len(remotes))
			for _, r := range remotes {
				fmt.Printf("  Name:      %s\n", r.Name)
				fmt.Printf("  URL:       %s\n", r.URL)
				fmt.Printf("  Directory: %s\n", r.Directory)

				if _, err := os.Stat(r.Directory); os.IsNotExist(err) {
					fmt.Printf("  Status:    Directory does not exist\n")
				} else {
					fmt.Printf("  Status:    Ready\n")
				}

				if r.LastPullAt != "" {
					fmt.Printf("  Last pull: %s\n", r.LastPullAt)
				}
				if r.LastPushAt != "" {
					fmt.Printf("  Last push: %s\n", r.LastPushAt)
				}
				fmt.Println()
			}

			return nil
		},
	}
	return cmd
}
