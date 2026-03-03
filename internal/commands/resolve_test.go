package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
	"github.com/rie/tasklean/internal/storage"
	"github.com/rie/tasklean/internal/testutil"
)

func TestResolve_NoConflicts(t *testing.T) {
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

	task := &models.Task{ID: "T-1", Title: "Clean task", Status: models.StatusTodo, Remote: name}
	if err := storage.SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	rc := NewRootCommand(config.DefaultConfig(), &mockTracker{})
	rc.SetArgs([]string{"resolve", name})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestResolve_ShowsConflictedTasks(t *testing.T) {
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

	task := &models.Task{ID: "CONF-1", Title: "Task with conflict", Status: models.StatusTodo, Remote: name}
	if err := storage.SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	if err := storage.WriteResolveBlock("CONF-1", dir, "changed in UI", "**Заголовок:** Task with conflict"); err != nil {
		t.Fatalf("WriteResolveBlock: %v", err)
	}

	rc := NewRootCommand(config.DefaultConfig(), &mockTracker{})
	rc.SetArgs([]string{"resolve", name})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestResolve_RemoteNotFound(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	rc := NewRootCommand(config.DefaultConfig(), &mockTracker{})
	rc.SetArgs([]string{"resolve", "nonexistent"})
	rc.SilenceUsage()

	err := rc.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent remote")
	}
	if !strings.Contains(err.Error(), `remote "nonexistent" not found`) {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPush_WritesResolveBlockOnSkip(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "origin"
	dir := filepath.Join(tmp, "tasks")
	if err := config.SaveRemoteConfigPlane(name, "https://task.example.com", "token", dir, "ws", "proj", false); err != nil {
		t.Fatalf("SaveRemoteConfigPlane: %v", err)
	}

	taskID := "SKIP-ME-1"
	localTime := time.Now().Add(-time.Hour).UTC()
	remoteTime := time.Now().UTC()

	// remote task is newer than local => push will skip
	tr := &pushMockTracker{
		fetched: []*models.Task{
			{ID: taskID, Title: "Remote version", Status: models.StatusTodo, Remote: name, UpdatedAt: remoteTime},
		},
	}

	localTask := &models.Task{
		ID:        taskID,
		Title:     "Local version",
		Status:    models.StatusTodo,
		Remote:    name,
		UpdatedAt: localTime,
	}
	if err := storage.SaveTask(localTask, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	// Set the index updated_at to localTime so push detects the skip condition.
	if err := storage.UpdateIndexUpdatedAt(taskID, dir, localTime.Format("2006-01-02T15:04:05Z")); err != nil {
		t.Fatalf("UpdateIndexUpdatedAt: %v", err)
	}

	rc := NewRootCommand(config.DefaultConfig(), tr)
	rc.SetArgs([]string{"push", name})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute push: %v", err)
	}

	// Verify resolve block was written to the .md file.
	data, err := os.ReadFile(filepath.Join(dir, "all", "todo.md"))
	if err != nil {
		t.Fatalf("read todo.md: %v", err)
	}
	if !strings.Contains(string(data), "### tasklean: resolve") {
		t.Errorf("expected resolve block written after skip, file content:\n%s", string(data))
	}
}
