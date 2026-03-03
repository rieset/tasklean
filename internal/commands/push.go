package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
	"github.com/rie/tasklean/internal/storage"
	"github.com/rie/tasklean/pkg/tracker"
	"github.com/spf13/cobra"
)

// modulesMatch returns true if two module names refer to the same module.
// Comparison is case-insensitive; spaces and dashes are treated as equivalent.
func modulesMatch(a, b string) bool {
	norm := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
	}
	return norm(a) == norm(b)
}

func moduleLabel(m string) string {
	if m == "" {
		return "(none)"
	}
	return m
}

func newPushCommand(tr tracker.Tracker) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <name>",
		Short: "Push tasks to remote tracker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(tr, args[0])
		},
	}
	return cmd
}

func runPush(tr tracker.Tracker, name string) error {
	remoteCfg, err := config.LoadRemoteConfig(name)
	if err != nil {
		return fmt.Errorf("remote %q not found: %w", name, err)
	}

	directory, err := resolveTaskDirectory(remoteCfg.Directory)
	if err != nil {
		return fmt.Errorf("resolve task directory: %w", err)
	}

	tasks, err := storage.ListTasks(directory)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	// Include tasks belonging to this remote and tasks with no remote set
	// (manually created in .md files without an _index.json entry).
	var toPush []*models.Task
	for _, t := range tasks {
		if t.Remote == name || t.Remote == "" {
			t.Remote = name
			toPush = append(toPush, t)
		}
	}

	if len(toPush) == 0 {
		fmt.Printf("No tasks to push for remote %q\n", name)
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		_ = config.UpdateRemoteLastPushAt(name, timestamp)
		return nil
	}

	ctx := context.Background()
	rl := newRateLimiter()

	var remoteTasks []*models.Task
	if err := rl.do(func() error {
		var e error
		remoteTasks, e = tr.FetchTasks(ctx, remoteCfg, nil)
		return e
	}); err != nil {
		return fmt.Errorf("fetch remote tasks: %w", err)
	}

	remoteByID := make(map[string]*models.Task)
	for _, t := range remoteTasks {
		remoteByID[t.ID] = t
	}

	total := len(toPush)
	fmt.Printf("Pushing %d tasks to %s...\n", total, name)

	updated, created, skipped, moduleSynced := 0, 0, 0, 0
	for i, t := range toPush {
		title := t.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		if remote, exists := remoteByID[t.ID]; exists {
			// Skip if task was changed in UI since our last pull (remote.updated_at > index.updated_at).
			// Compare at second granularity to avoid false skips from fractional seconds in API.
			remoteSec := remote.UpdatedAt.Truncate(time.Second)
			localSec := t.UpdatedAt.Truncate(time.Second)
			if !t.UpdatedAt.IsZero() && remoteSec.After(localSec) {
				fmt.Printf("[%d/%d] skip      %s (changed in UI)\n", i+1, total, title)
				_ = storage.WriteResolveBlock(t.ID, directory,
					"задача изменена в UI (remote версия новее локальной)",
					localTaskSnapshot(t),
				)
				skipped++
				continue
			}

			fmt.Printf("[%d/%d] updating  %s\n", i+1, total, title)
			rl.wait()
			if err := rl.do(func() error { return tr.UpdateTask(ctx, remoteCfg, t) }); err != nil {
				return fmt.Errorf("update task %s: %w", t.ID, err)
			}
			// Update index so next push won't skip (Plane sets updated_at to now)
			_ = storage.UpdateIndexUpdatedAt(t.ID, directory, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
			updated++

			// Only sync module when local module is explicitly set.
			// If local module is empty, leave the Plane module unchanged.
			if t.Module != "" && !modulesMatch(remote.Module, t.Module) {
				fmt.Printf("[%d/%d] module    %s → %s\n", i+1, total, moduleLabel(remote.Module), moduleLabel(t.Module))
				rl.wait()
				if err := rl.do(func() error {
					return tr.SyncTaskModule(ctx, remoteCfg, t.ID, remote.Module, t.Module)
				}); err != nil {
					fmt.Printf("         warn: module sync failed: %v\n", err)
				} else {
					moduleSynced++
				}
			}
		} else {
			fmt.Printf("[%d/%d] creating  %s\n", i+1, total, title)
			t.Remote = name
			var createdTask *models.Task
			rl.wait()
			if err := rl.do(func() error {
				var e error
				createdTask, e = tr.CreateTask(ctx, remoteCfg, t)
				return e
			}); err != nil {
				if strings.Contains(err.Error(), "429") {
					return fmt.Errorf("create task %q: %w (Plane rate limit; try again in 1–2 minutes or increase API_KEY_RATE_LIMIT on server)", t.Title, err)
				}
				return fmt.Errorf("create task %q: %w", t.Title, err)
			}
			if err := storage.ReplaceTaskID(t.ID, createdTask, directory); err != nil {
				return fmt.Errorf("replace task ID %s: %w", t.ID, err)
			}
			if t.Module != "" {
				rl.wait()
				if err := rl.do(func() error {
					return tr.SyncTaskModule(ctx, remoteCfg, createdTask.ID, "", t.Module)
				}); err != nil {
					fmt.Printf("         warn: module sync failed: %v\n", err)
				} else {
					moduleSynced++
				}
			}
			created++
		}
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if err := config.UpdateRemoteLastPushAt(name, timestamp); err != nil {
		return fmt.Errorf("update last push time: %w", err)
	}

	summary := fmt.Sprintf("\nDone: %d updated, %d created", updated, created)
	if skipped > 0 {
		summary += fmt.Sprintf(", %d skipped (changed in UI)", skipped)
	}
	if moduleSynced > 0 {
		summary += fmt.Sprintf(", %d module(s) synced", moduleSynced)
	}
	fmt.Println(summary)
	return nil
}

// localTaskSnapshot builds a markdown-formatted snapshot of the local task fields
// to embed in the resolve block so the user can see what they were trying to push.
func localTaskSnapshot(t *models.Task) string {
	var sb strings.Builder
	sb.WriteString("**Заголовок:** " + t.Title)
	sb.WriteString("\n\n**Статус:** " + string(t.Status))
	if t.Module != "" {
		sb.WriteString("\n\n**Модуль:** " + t.Module)
	}
	if t.Description != "" {
		sb.WriteString("\n\n**Описание:**\n\n" + t.Description)
	}
	return sb.String()
}
