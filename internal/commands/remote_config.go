package commands

import (
	"fmt"

	"github.com/rie/tasklean/internal/config"
	"github.com/spf13/cobra"
)

func newRemoteConfigCommand() *cobra.Command {
	var workspace string
	var project string
	var cloud bool
	var noCloud bool

	cmd := &cobra.Command{
		Use:   "config <name>",
		Short: "Update remote configuration (workspace, project, cloud mode)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.LoadRemoteConfig(name)
			if err != nil {
				return fmt.Errorf("remote %q not found: %w", name, err)
			}
			if workspace != "" {
				cfg.Workspace = workspace
			}
			if project != "" {
				cfg.Project = project
			}
			if cloud {
				cfg.Cloud = true
			}
			if noCloud {
				cfg.Cloud = false
			}
			if err := config.SaveRemoteConfigFromStruct(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			cloudMode := "self-hosted"
			if cfg.Cloud {
				cloudMode = "cloud"
			}
			fmt.Printf("Updated remote %q (workspace=%q project=%q mode=%s)\n",
				name, cfg.Workspace, cfg.Project, cloudMode)
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Plane workspace slug")
	cmd.Flags().StringVarP(&project, "project", "p", "", "Plane project ID")
	cmd.Flags().BoolVar(&cloud, "cloud", false, "use Plane Cloud API (work-items endpoint)")
	cmd.Flags().BoolVar(&noCloud, "no-cloud", false, "use self-hosted Plane API (issues endpoint)")
	return cmd
}
