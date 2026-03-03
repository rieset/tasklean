package commands

import (
	"fmt"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/storage"
	"github.com/spf13/cobra"
)

func newResolveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <name>",
		Short: "List tasks with unresolved push conflicts",
		Long: `Shows tasks that were skipped during push because they were changed in the UI.
Each listed task has a "### tasklean: resolve" block in its .md file.
Resolve conflicts manually: run pull to fetch the latest remote version, merge
your local changes if needed, then delete the resolve block from the .md file.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResolve(args[0])
		},
	}
}

func runResolve(name string) error {
	remoteCfg, err := config.LoadRemoteConfig(name)
	if err != nil {
		return fmt.Errorf("remote %q not found: %w", name, err)
	}

	directory, err := resolveTaskDirectory(remoteCfg.Directory)
	if err != nil {
		return fmt.Errorf("resolve task directory: %w", err)
	}

	tasks, err := storage.ListResolvedTasks(directory)
	if err != nil {
		return fmt.Errorf("list resolved tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks with resolve block.")
		return nil
	}

	fmt.Printf("Tasks with unresolved conflicts (%d):\n\n", len(tasks))
	for _, t := range tasks {
		fmt.Printf("  [%s] %s\n", t.ID, t.Title)
		fmt.Printf("  %s\n", t.File)
		if t.ResolveText != "" {
			fmt.Printf("  ↳ %s\n", t.ResolveText)
		}
		fmt.Println()
	}
	fmt.Println("To resolve: edit the .md file, remove the \"### tasklean: resolve\" block, then push again.")
	return nil
}
