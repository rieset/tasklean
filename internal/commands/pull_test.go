package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
	"github.com/rie/tasklean/internal/storage"
	"github.com/rie/tasklean/internal/testutil"
	"github.com/rie/tasklean/pkg/tracker"
)

type mockTracker struct {
	tasks    []*models.Task
	err      error
	lastOpts *tracker.FetchOptions
}

func (m *mockTracker) FetchTasks(_ context.Context, _ *config.RemoteConfig, opts *tracker.FetchOptions) ([]*models.Task, error) {
	m.lastOpts = opts
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}

func (m *mockTracker) UpdateTask(_ context.Context, _ *config.RemoteConfig, _ *models.Task) error {
	return nil
}

func (m *mockTracker) CreateTask(_ context.Context, _ *config.RemoteConfig, _ *models.Task) (*models.Task, error) {
	return nil, tracker.ErrNotImplemented
}

func (m *mockTracker) SyncTaskModule(_ context.Context, _ *config.RemoteConfig, _, _, _ string) error {
	return nil
}

func TestPull_SavesTasksAsFiles(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "origin"
	dir := filepath.Join(tmp, "tasks")
	if err := config.SaveRemoteConfig(name, "https://task.example.com", "token", dir); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	now := time.Now().UTC()
	tr := &mockTracker{
		tasks: []*models.Task{
			{
				ID:          "TASK-1",
				Title:       "First task",
				Description: "Description one",
				Status:      models.StatusTodo,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          "TASK-2",
				Title:       "Second task",
				Description: "Description two",
				Status:      models.StatusInProgress,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}

	rc := NewRootCommand(config.DefaultConfig(), tr)
	rc.SetArgs([]string{"pull", name})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	tasks, err := storage.ListTasks(dir)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("ListTasks: got %d tasks, want 2", len(tasks))
	}

	if _, err := os.Stat(filepath.Join(dir, "all", "todo.md")); os.IsNotExist(err) {
		t.Errorf("all/todo.md was not created")
	}
	if _, err := os.Stat(filepath.Join(dir, "all", "in_progress.md")); os.IsNotExist(err) {
		t.Errorf("all/in_progress.md was not created")
	}

	loaded, err := storage.LoadTask("TASK-1", dir)
	if err != nil {
		t.Fatalf("LoadTask TASK-1: %v", err)
	}
	if loaded.Title != "First task" {
		t.Errorf("Title = %q, want %q", loaded.Title, "First task")
	}
	if loaded.Description != "Description one" {
		t.Errorf("Description = %q, want %q", loaded.Description, "Description one")
	}
	if loaded.Remote != name {
		t.Errorf("Remote = %q, want %q", loaded.Remote, name)
	}

	cfg, err := config.LoadRemoteConfig(name)
	if err != nil {
		t.Fatalf("LoadRemoteConfig: %v", err)
	}
	if cfg.LastPullAt == "" {
		t.Error("LastPullAt should be set after pull")
	}
}

func TestPull_RemoteNotFound(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	rc := NewRootCommand(config.DefaultConfig(), &mockTracker{})
	rc.SetArgs([]string{"pull", "nonexistent"})
	rc.SilenceUsage()

	err := rc.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent remote")
	}
	if !strings.Contains(err.Error(), `remote "nonexistent" not found`) {
		t.Errorf("error should mention remote not found: %v", err)
	}
	if !strings.Contains(err.Error(), "failed to read config") {
		t.Errorf("error should mention config: %v", err)
	}
}

func TestPull_TrackerError(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "origin"
	if err := config.SaveRemoteConfig(name, "https://x.com", "token", "."); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	tr := &mockTracker{err: tracker.ErrNotImplemented}

	rc := NewRootCommand(config.DefaultConfig(), tr)
	rc.SetArgs([]string{"pull", name})
	rc.SilenceUsage()

	err := rc.Execute()
	if err == nil {
		t.Fatal("expected error when tracker fails")
	}
	if err.Error() != "failed to fetch tasks: tracker API not implemented" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPull_PassesFlagsToTracker(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "origin"
	if err := config.SaveRemoteConfig(name, "https://x.com", "token", "."); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	tr := &mockTracker{tasks: []*models.Task{}}

	rc := NewRootCommand(config.DefaultConfig(), tr)
	rc.SetArgs([]string{"pull", name, "--from", "2024-06-01", "--assignee", "user@example.com"})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if tr.lastOpts == nil {
		t.Fatal("FetchTasks should receive options")
	}
	if tr.lastOpts.From != "2024-06-01" {
		t.Errorf("From = %q, want 2024-06-01", tr.lastOpts.From)
	}
	if tr.lastOpts.Assignee != "user@example.com" {
		t.Errorf("Assignee = %q, want user@example.com", tr.lastOpts.Assignee)
	}
}

func TestPull_ShortFlags(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "origin"
	if err := config.SaveRemoteConfig(name, "https://x.com", "token", "."); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	tr := &mockTracker{tasks: []*models.Task{}}

	rc := NewRootCommand(config.DefaultConfig(), tr)
	rc.SetArgs([]string{"pull", name, "-f", "2024-01-15", "-a", "dev@company.org"})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if tr.lastOpts.From != "2024-01-15" {
		t.Errorf("From (-f) = %q, want 2024-01-15", tr.lastOpts.From)
	}
	if tr.lastOpts.Assignee != "dev@company.org" {
		t.Errorf("Assignee (-a) = %q, want dev@company.org", tr.lastOpts.Assignee)
	}
}
