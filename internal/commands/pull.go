package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/storage"
	"github.com/rie/tasklean/pkg/tracker"
	"github.com/spf13/cobra"
)

func newPullCommand(tr tracker.Tracker) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <name>",
		Short: "Pull tasks from remote tracker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			from, _ := cmd.Flags().GetString("from")
			assignee, _ := cmd.Flags().GetString("assignee")
			debug, _ := cmd.Flags().GetBool("debug")
			opts := &tracker.FetchOptions{From: from, Assignee: assignee, Debug: debug}
			return runPull(tr, args[0], opts, debug)
		},
	}
	cmd.Flags().StringP("from", "f", "", "filter tasks updated from date (YYYY-MM-DD)")
	cmd.Flags().StringP("assignee", "a", "", "filter by assignee email")
	cmd.Flags().Bool("debug", false, "print first 3 tasks with module info for diagnostics")
	return cmd
}

func runPull(tr tracker.Tracker, name string, opts *tracker.FetchOptions, debug bool) error {
	remoteCfg, err := config.LoadRemoteConfig(name)
	if err != nil {
		return fmt.Errorf("remote %q not found: %w", name, err)
	}

	ctx := context.Background()
	tasks, err := tr.FetchTasks(ctx, remoteCfg, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	if debug {
		n := 3
		if len(tasks) < n {
			n = len(tasks)
		}
		for i := 0; i < n; i++ {
			t := tasks[i]
			fmt.Printf("[debug] task %s: title=%q module=%q status=%s\n", t.ID, t.Title, t.Module, t.Status)
		}
	}

	directory, err := resolveTaskDirectory(remoteCfg.Directory)
	if err != nil {
		return fmt.Errorf("failed to resolve task directory: %w", err)
	}

	for _, task := range tasks {
		task.Remote = name
		if err := storage.SaveTask(task, directory); err != nil {
			return fmt.Errorf("failed to save task %s: %w", task.ID, err)
		}
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if err := config.UpdateRemoteLastPullAt(name, timestamp); err != nil {
		return fmt.Errorf("failed to update last pull time: %w", err)
	}

	return nil
}

func resolveTaskDirectory(dir string) (string, error) {
	if dir == "" || dir == "." {
		return ".", nil
	}
	return filepath.Abs(dir)
}
