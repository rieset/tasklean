package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rie/tasklean/internal/models"
)

func TestExtractResolveBlock(t *testing.T) {
	body := "## Title\n\nDescription.\n\n### tasklean: resolve\n\nПропущено при push: 2026-01-01T00:00:00Z — причина."

	got := ExtractResolveBlock(body)
	if !strings.HasPrefix(got, "### tasklean: resolve") {
		t.Errorf("expected resolve header, got: %q", got)
	}
	if !strings.Contains(got, "Пропущено при push") {
		t.Errorf("expected resolve body, got: %q", got)
	}
}

func TestExtractResolveBlock_Missing(t *testing.T) {
	body := "## Title\n\nDescription without resolve block."
	if got := ExtractResolveBlock(body); got != "" {
		t.Errorf("expected empty, got: %q", got)
	}
}

func TestExtractResolveBlock_BodyIsOnlyHeader(t *testing.T) {
	body := "### tasklean: resolve\n\nsome text"
	got := ExtractResolveBlock(body)
	if !strings.HasPrefix(got, "### tasklean: resolve") {
		t.Errorf("expected resolve header when body starts with it, got: %q", got)
	}
}

func TestStripResolveBlock(t *testing.T) {
	body := "## Title\n\nDescription.\n\n### tasklean: resolve\n\nConflict info."
	got := StripResolveBlock(body)
	if strings.Contains(got, "### tasklean: resolve") {
		t.Errorf("expected resolve block stripped, got: %q", got)
	}
	if !strings.Contains(got, "## Title") {
		t.Errorf("expected title preserved, got: %q", got)
	}
	if !strings.Contains(got, "Description.") {
		t.Errorf("expected description preserved, got: %q", got)
	}
}

func TestStripResolveBlock_NoBlock(t *testing.T) {
	body := "## Title\n\nDescription."
	got := StripResolveBlock(body)
	if got != body {
		t.Errorf("expected unchanged body, got: %q", got)
	}
}

func TestMergeResolveBlock(t *testing.T) {
	newBody := "## New Title\n\nNew description."
	resolveBlock := "### tasklean: resolve\n\nConflict info."

	got := MergeResolveBlock(newBody, resolveBlock)
	if !strings.Contains(got, "## New Title") {
		t.Errorf("expected new body in result, got: %q", got)
	}
	if !strings.Contains(got, resolveBlock) {
		t.Errorf("expected resolve block in result, got: %q", got)
	}
	// resolve block must come after description
	if strings.Index(got, "New description.") > strings.Index(got, resolveBlockHeader) {
		t.Errorf("expected resolve block after new body")
	}
}

func TestMergeResolveBlock_EmptyResolve(t *testing.T) {
	newBody := "## Title\n\nDesc."
	got := MergeResolveBlock(newBody, "")
	if got != newBody {
		t.Errorf("expected unchanged body when resolve is empty, got: %q", got)
	}
}

func TestWriteResolveBlock(t *testing.T) {
	dir := t.TempDir()
	task := &models.Task{
		ID:          "TASK-RES-1",
		Title:       "Resolve task",
		Description: "Some description.",
		Status:      models.StatusTodo,
	}
	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	if err := WriteResolveBlock("TASK-RES-1", dir, "тест", "**Заголовок:** Resolve task\n\n**Описание:**\n\nSome description."); err != nil {
		t.Fatalf("WriteResolveBlock: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "all", "todo.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, resolveBlockHeader) {
		t.Errorf("expected resolve block in file, got:\n%s", content)
	}
	if !strings.Contains(content, "тест") {
		t.Errorf("expected reason in resolve block, got:\n%s", content)
	}
}

func TestWriteResolveBlock_UpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	task := &models.Task{
		ID:     "TASK-RES-2",
		Title:  "Update resolve",
		Status: models.StatusTodo,
	}
	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	if err := WriteResolveBlock("TASK-RES-2", dir, "первая причина", "**Заголовок:** v1"); err != nil {
		t.Fatalf("first WriteResolveBlock: %v", err)
	}
	if err := WriteResolveBlock("TASK-RES-2", dir, "вторая причина", "**Заголовок:** v2"); err != nil {
		t.Fatalf("second WriteResolveBlock: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "all", "todo.md"))
	content := string(data)
	count := strings.Count(content, resolveBlockHeader)
	if count != 1 {
		t.Errorf("expected exactly one resolve block header, got %d:\n%s", count, content)
	}
	if !strings.Contains(content, "вторая причина") {
		t.Errorf("expected updated reason in resolve block, got:\n%s", content)
	}
	if strings.Contains(content, "первая причина") {
		t.Errorf("expected old reason replaced, got:\n%s", content)
	}
}

func TestListResolvedTasks(t *testing.T) {
	dir := t.TempDir()

	tasks := []*models.Task{
		{ID: "RES-1", Title: "Conflicted task", Status: models.StatusTodo},
		{ID: "RES-2", Title: "Clean task", Status: models.StatusTodo},
	}
	for _, task := range tasks {
		if err := SaveTask(task, dir); err != nil {
			t.Fatalf("SaveTask: %v", err)
		}
	}

	if err := WriteResolveBlock("RES-1", dir, "конфликт", ""); err != nil {
		t.Fatalf("WriteResolveBlock: %v", err)
	}

	resolved, err := ListResolvedTasks(dir)
	if err != nil {
		t.Fatalf("ListResolvedTasks: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved task, got %d", len(resolved))
	}
	if resolved[0].ID != "RES-1" {
		t.Errorf("expected RES-1, got %s", resolved[0].ID)
	}
	if resolved[0].Title != "Conflicted task" {
		t.Errorf("unexpected title: %s", resolved[0].Title)
	}
	if resolved[0].File == "" {
		t.Error("expected non-empty file path")
	}
	if resolved[0].ResolveText == "" {
		t.Error("expected non-empty resolve text")
	}
}

func TestSaveTask_PreservesResolveBlock(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{
		ID:          "RES-PULL-1",
		Title:       "Task with conflict",
		Description: "Original description.",
		Status:      models.StatusTodo,
	}
	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	if err := WriteResolveBlock("RES-PULL-1", dir, "changed in UI", "**Заголовок:** Task with conflict"); err != nil {
		t.Fatalf("WriteResolveBlock: %v", err)
	}

	// Simulate pull: save updated version from remote.
	updated := &models.Task{
		ID:          "RES-PULL-1",
		Title:       "Task with conflict (updated from remote)",
		Description: "Description updated by remote.",
		Status:      models.StatusTodo,
	}
	if err := SaveTask(updated, dir); err != nil {
		t.Fatalf("SaveTask (pull update): %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "all", "todo.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "updated from remote") {
		t.Errorf("expected updated title from remote, got:\n%s", content)
	}
	if !strings.Contains(content, resolveBlockHeader) {
		t.Errorf("expected resolve block preserved after pull, got:\n%s", content)
	}
	if !strings.Contains(content, "changed in UI") {
		t.Errorf("expected resolve reason preserved after pull, got:\n%s", content)
	}
}

func TestListResolvedTasks_Empty(t *testing.T) {
	dir := t.TempDir()

	task := &models.Task{ID: "NO-RESOLVE-1", Title: "Clean", Status: models.StatusTodo}
	if err := SaveTask(task, dir); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	resolved, err := ListResolvedTasks(dir)
	if err != nil {
		t.Fatalf("ListResolvedTasks: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved tasks, got %d", len(resolved))
	}
}
