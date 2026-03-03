package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rie/tasklean/internal/models"
)

func TestSaveAndLoadTask(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:          "TEST-001",
		Title:       "Test Task",
		Description: "Test description",
		Status:      models.StatusInProgress,
		Remote:      "origin",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	loaded, err := LoadTask("TEST-001", dir)
	if err != nil {
		t.Fatalf("Failed to load task: %v", err)
	}

	if loaded.ID != task.ID {
		t.Errorf("ID mismatch: got %v, want %v", loaded.ID, task.ID)
	}
	if loaded.Title != task.Title {
		t.Errorf("Title mismatch: got %v, want %v", loaded.Title, task.Title)
	}
	if loaded.Description != task.Description {
		t.Errorf("Description mismatch: got %v, want %v", loaded.Description, task.Description)
	}
	if loaded.Status != task.Status {
		t.Errorf("Status mismatch: got %v, want %v", loaded.Status, task.Status)
	}
}

func TestListTasks(t *testing.T) {
	dir := t.TempDir()

	tasks := []*models.Task{
		{ID: "TASK-001", Title: "Task 1", Status: models.StatusTodo},
		{ID: "TASK-002", Title: "Task 2", Status: models.StatusDone},
		{ID: "TASK-003", Title: "Task 3", Status: models.StatusBacklog},
	}

	for _, task := range tasks {
		if err := SaveTask(task, dir); err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}
	}

	loaded, err := ListTasks(dir)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(loaded))
	}

	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Readme"), 0644)

	loaded, err = ListTasks(dir)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	if len(loaded) != 3 {
		t.Errorf("Expected 3 tasks (README.md ignored), got %d", len(loaded))
	}
}

func TestDeleteTask(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:     "TEST-001",
		Title:  "Test Task",
		Status: models.StatusTodo,
	}

	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	if err := DeleteTask("TEST-001", dir); err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	if _, err := LoadTask("TEST-001", dir); err == nil {
		t.Error("Task should be deleted")
	}
}

func TestSaveTask_CreatesStatusFile(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:          "TASK-1",
		Title:       "My task",
		Description: "Desc",
		Status:      models.StatusTodo,
		Remote:      "origin",
	}

	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "all", "todo.md"))
	if err != nil {
		t.Fatalf("Read all/todo.md: %v", err)
	}
	if !strings.Contains(string(data), "id:TASK-1") {
		t.Errorf("todo.md should contain id:TASK-1")
	}
	if !strings.Contains(string(data), "📋") {
		t.Errorf("todo.md should contain status emoji")
	}
	if !strings.HasPrefix(string(data), "# Todo") {
		preview := string(data)
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		t.Errorf("todo.md should start with H1 status name, got: %s", preview)
	}
	if !strings.Contains(string(data), "📥 backlog | 📋 todo") {
		t.Errorf("todo.md should contain legend")
	}
	if !strings.Contains(string(data), "## My task") {
		t.Errorf("todo.md should contain task title as markdown")
	}

	idxData, err := os.ReadFile(filepath.Join(dir, indexFilename))
	if err != nil {
		t.Fatalf("Read index: %v", err)
	}
	if !strings.Contains(string(idxData), "origin") {
		t.Errorf("index should contain remote")
	}
}

func TestSaveTask_PreservesMarkdown(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:          "TASK-MD",
		Title:       "Task with **bold**",
		Description: "Description with\n- list\n- items\n\nAnd `code`.",
		Status:      models.StatusTodo,
	}

	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	loaded, err := LoadTask("TASK-MD", dir)
	if err != nil {
		t.Fatalf("LoadTask: %v", err)
	}
	if loaded.Title != "Task with **bold**" {
		t.Errorf("Title: got %q", loaded.Title)
	}
	if !strings.Contains(loaded.Description, "- list") || !strings.Contains(loaded.Description, "`code`") {
		t.Errorf("Description should preserve markdown: got %q", loaded.Description)
	}
}

func TestListTasksEmptyDir(t *testing.T) {
	dir := t.TempDir()

	tasks, err := ListTasks(dir)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestListTasksNonExistent(t *testing.T) {
	tasks, err := ListTasks("/nonexistent/path")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestSaveTask_WithModule(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:     "TASK-MOD",
		Title:  "Module task",
		Status: models.StatusTodo,
		Module: "Backend API",
		Remote: "origin",
	}

	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "all", "todo-backend-api.md")); os.IsNotExist(err) {
		t.Errorf("all/todo-backend-api.md should exist")
	}

	loaded, err := LoadTask("TASK-MOD", dir)
	if err != nil {
		t.Fatalf("LoadTask: %v", err)
	}
	if loaded.Module != "Backend API" {
		t.Errorf("Module: got %q, want Backend API", loaded.Module)
	}
}

func TestSaveTask_AssigneeFolders(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:        "TASK-ASSIGN",
		Title:     "Assigned task",
		Status:    models.StatusTodo,
		Assignees: []string{"user1@example.com", "user2@test.com"},
		Remote:    "origin",
	}

	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	// sanitize: user@example.com -> user-at-example.com (dot kept)
	a1 := filepath.Join(dir, "user1-at-example.com", "todo.md")
	a2 := filepath.Join(dir, "user2-at-test.com", "todo.md")
	if _, err := os.Stat(a1); os.IsNotExist(err) {
		t.Errorf("assignee1 todo.md should exist at %s", a1)
	}
	if _, err := os.Stat(a2); os.IsNotExist(err) {
		t.Errorf("assignee2 todo.md should exist at %s", a2)
	}
}

func TestSaveTask_UnassignedGoesToAll(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:        "TASK-NO-ASSIGN",
		Title:     "Unassigned task",
		Status:    models.StatusTodo,
		Assignees: nil,
		Remote:    "origin",
	}

	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	allPath := filepath.Join(dir, "all", "todo.md")
	if _, err := os.Stat(allPath); os.IsNotExist(err) {
		t.Errorf("unassigned task should exist at all/todo.md")
	}
}

func TestLoadTask_EmojiOverridesStatus(t *testing.T) {
	dir := t.TempDir()

	content := `[✅ id:TASK-DONE]

## Task marked done

Description
`
	if err := os.WriteFile(filepath.Join(dir, "todo.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Write: %v", err)
	}
	idxContent := `{"TASK-DONE":{"remote":"origin","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}}`
	if err := os.WriteFile(filepath.Join(dir, "_index.json"), []byte(idxContent), 0644); err != nil {
		t.Fatalf("Write index: %v", err)
	}

	loaded, err := LoadTask("TASK-DONE", dir)
	if err != nil {
		t.Fatalf("LoadTask: %v", err)
	}
	if loaded.Status != models.StatusDone {
		t.Errorf("Status from emoji: got %v, want done", loaded.Status)
	}
}
