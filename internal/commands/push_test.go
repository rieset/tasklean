package commands

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
	"github.com/rie/tasklean/internal/storage"
	"github.com/rie/tasklean/internal/testutil"
	"github.com/rie/tasklean/pkg/tracker"
)

type pushMockTracker struct {
	fetched   []*models.Task
	updated   []string
	created   []*models.Task
	updateErr error
	createErr error
}

func (m *pushMockTracker) FetchTasks(_ context.Context, _ *config.RemoteConfig, _ *tracker.FetchOptions) ([]*models.Task, error) {
	return m.fetched, nil
}

func (m *pushMockTracker) UpdateTask(_ context.Context, _ *config.RemoteConfig, task *models.Task) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, task.ID)
	return nil
}

func (m *pushMockTracker) CreateTask(_ context.Context, _ *config.RemoteConfig, task *models.Task) (*models.Task, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	created := &models.Task{
		ID:          "created-uuid-123",
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Remote:      task.Remote,
		Assignees:   task.Assignees,
	}
	m.created = append(m.created, created)
	return created, nil
}

func (m *pushMockTracker) SyncTaskModule(_ context.Context, _ *config.RemoteConfig, _, _, _ string) error {
	return nil
}

func TestPush_UpdatesAndCreates(t *testing.T) {
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

	existingID := "existing-uuid-456"
	tr := &pushMockTracker{
		fetched: []*models.Task{
			{ID: existingID, Title: "Existing", Status: models.StatusTodo, Remote: name},
		},
	}

	task1 := &models.Task{ID: existingID, Title: "Updated title", Description: "Updated desc", Status: models.StatusInProgress, Remote: name}
	if err := storage.SaveTask(task1, dir); err != nil {
		t.Fatalf("SaveTask existing: %v", err)
	}

	task2 := &models.Task{ID: "new-local-id", Title: "New task", Description: "New desc", Status: models.StatusTodo, Remote: name}
	if err := storage.SaveTask(task2, dir); err != nil {
		t.Fatalf("SaveTask new: %v", err)
	}

	rc := NewRootCommand(config.DefaultConfig(), tr)
	rc.SetArgs([]string{"push", name})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(tr.updated) != 1 || tr.updated[0] != existingID {
		t.Errorf("UpdateTask: got %v, want [%s]", tr.updated, existingID)
	}
	if len(tr.created) != 1 || tr.created[0].Title != "New task" {
		t.Errorf("CreateTask: got %v", tr.created)
	}

	cfg, _ := config.LoadRemoteConfig(name)
	if cfg.LastPushAt == "" {
		t.Error("LastPushAt should be set after push")
	}
}
