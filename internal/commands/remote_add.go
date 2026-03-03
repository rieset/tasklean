package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/tui"
	"github.com/spf13/cobra"
)

func newRemoteAddCommand() *cobra.Command {
	var directory string
	var token string
	var skipValidation bool
	var workspace string
	var project string
	var cloud bool

	cmd := &cobra.Command{
		Use:   "add <name> [project-url]",
		Short: "Add a remote tracker",
		Long:  "Add a new remote tracker connection. Provide a link to the project issues list to auto-detect workspace and project.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			projectURL := ""
			if len(args) > 1 {
				projectURL = args[1]
			}
			if projectURL == "" {
				u, err := tui.PromptText("Link to project issues:", "e.g. https://your-tracker.example.com/workspace/projects/uuid/issues/")
				if err != nil {
					return fmt.Errorf("URL input cancelled: %w", err)
				}
				projectURL = u
			}
			if projectURL == "" {
				return fmt.Errorf("project URL cannot be empty")
			}

			if config.RemoteConfigExists(name) {
				return fmt.Errorf("remote %q already exists", name)
			}

			remoteURL := projectURL
			if base, ws, prj, ok := config.ParsePlaneIssuesURL(projectURL); ok {
				remoteURL = base
				if workspace == "" {
					workspace = ws
				}
				if project == "" {
					project = prj
				}
				if !cloud && strings.Contains(projectURL, "app.plane.so") {
					cloud = true
				}
			} else if _, err := url.Parse(projectURL); err != nil {
				return fmt.Errorf("invalid URL (expected project issues link): %w", err)
			}

			if directory == "" {
				dir, err := tui.PromptText("Directory for tasks:", "e.g. ./tasks")
				if err != nil {
					return fmt.Errorf("directory input cancelled: %w", err)
				}
				directory = dir
			}
			if directory == "" {
				directory = "."
			}

			if token == "" {
				tuiToken, err := tui.PromptToken("Enter API token:")
				if err != nil {
					return fmt.Errorf("token input cancelled: %w", err)
				}
				token = tuiToken
			}

			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			if workspace == "" {
				ws, err := tui.PromptText("Workspace slug:", "e.g. my-workspace")
				if err != nil {
					return fmt.Errorf("workspace input cancelled: %w", err)
				}
				workspace = ws
			}

			if project == "" {
				prj, err := tui.PromptText("Project ID (UUID or identifier):", "e.g. 3481d8a2-bcdc-4553-aa8c-922bcf009255")
				if err != nil {
					return fmt.Errorf("project input cancelled: %w", err)
				}
				project = prj
			}

			if !skipValidation {
				fmt.Printf("Validating connection to %s... ", remoteURL)
				client := &http.Client{Timeout: 10 * time.Second}
				req, err := http.NewRequest("GET", remoteURL, nil)
				if err == nil {
					req.Header.Set("Authorization", "Bearer "+token)
					resp, err := client.Do(req)
					if err == nil {
						resp.Body.Close()
						if resp.StatusCode < 400 {
							fmt.Println("OK")
						} else {
							fmt.Printf("Warning: server returned status %d (use -s to skip validation)\n", resp.StatusCode)
						}
					} else {
						fmt.Printf("Warning: %v (use -s to skip validation)\n", err)
					}
				}
			}

			if err := config.SaveRemoteConfigPlane(name, remoteURL, token, directory, workspace, project, cloud); err != nil {
				return fmt.Errorf("failed to save remote config: %w", err)
			}

			fmt.Printf("Added remote %q at %q\n", name, remoteURL)
			fmt.Printf("Tasks will be stored in: %s\n", directory)
			return nil
		},
	}

	cmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory to store tasks (default: current directory)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "API token (will prompt if not provided)")
	cmd.Flags().BoolVarP(&skipValidation, "skip-validation", "s", false, "Skip connection validation")
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "workspace slug (overrides value from URL)")
	cmd.Flags().StringVarP(&project, "project", "p", "", "project ID (overrides value from URL)")
	cmd.Flags().BoolVar(&cloud, "cloud", false, "use cloud API (work-items endpoint)")

	return cmd
}
