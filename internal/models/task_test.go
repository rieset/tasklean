package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{StatusBacklog, true},
		{StatusTodo, true},
		{StatusInProgress, true},
		{StatusInReview, true},
		{StatusDone, true},
		{StatusCancelled, true},
		{TaskStatus("invalid"), false},
		{TaskStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("TaskStatus.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTask_JSON(t *testing.T) {
	task := Task{
		ID:          "TASK-123",
		Title:       "Test Task",
		Description: "Test description",
		Status:      StatusInProgress,
		Remote:      "origin",
		CreatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var unmarshaled Task
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if unmarshaled.ID != task.ID {
		t.Errorf("Task ID mismatch: got %v, want %v", unmarshaled.ID, task.ID)
	}
	if unmarshaled.Title != task.Title {
		t.Errorf("Task Title mismatch: got %v, want %v", unmarshaled.Title, task.Title)
	}
	if unmarshaled.Description != task.Description {
		t.Errorf("Task Description mismatch: got %v, want %v", unmarshaled.Description, task.Description)
	}
	if unmarshaled.Status != task.Status {
		t.Errorf("Task Status mismatch: got %v, want %v", unmarshaled.Status, task.Status)
	}
	if unmarshaled.Remote != task.Remote {
		t.Errorf("Task Remote mismatch: got %v, want %v", unmarshaled.Remote, task.Remote)
	}
	if !unmarshaled.CreatedAt.Equal(task.CreatedAt) {
		t.Errorf("Task CreatedAt mismatch: got %v, want %v", unmarshaled.CreatedAt, task.CreatedAt)
	}
	if !unmarshaled.UpdatedAt.Equal(task.UpdatedAt) {
		t.Errorf("Task UpdatedAt mismatch: got %v, want %v", unmarshaled.UpdatedAt, task.UpdatedAt)
	}
}

func TestSyncStatus_MarkPulled(t *testing.T) {
	status := NewSyncStatus("origin")
	before := time.Now()
	status.MarkPulled()
	after := time.Now()

	if status.LastPullAt == nil {
		t.Fatal("LastPullAt should not be nil")
	}
	if status.LastPullAt.Before(before) || status.LastPullAt.After(after) {
		t.Errorf("LastPullAt not in expected range: %v", status.LastPullAt)
	}
}

func TestSyncStatus_MarkPushed(t *testing.T) {
	status := NewSyncStatus("origin")
	before := time.Now()
	status.MarkPushed()
	after := time.Now()

	if status.LastPushAt == nil {
		t.Fatal("LastPushAt should not be nil")
	}
	if status.LastPushAt.Before(before) || status.LastPushAt.After(after) {
		t.Errorf("LastPushAt not in expected range: %v", status.LastPushAt)
	}
}

func TestSyncStatus_JSON(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	status := &SyncStatus{
		RemoteName:    "origin",
		Direction:     SyncDirectionBoth,
		LastPullAt:    &now,
		LastPushAt:    &now,
		TasksTotal:    15,
		TasksDone:     5,
		LocalChanges:  2,
		RemoteChanges: 0,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal SyncStatus: %v", err)
	}

	var unmarshaled SyncStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal SyncStatus: %v", err)
	}

	if unmarshaled.RemoteName != status.RemoteName {
		t.Errorf("RemoteName: got %v, want %v", unmarshaled.RemoteName, status.RemoteName)
	}
	if unmarshaled.Direction != status.Direction {
		t.Errorf("Direction: got %v, want %v", unmarshaled.Direction, status.Direction)
	}
	if unmarshaled.TasksTotal != status.TasksTotal {
		t.Errorf("TasksTotal: got %v, want %v", unmarshaled.TasksTotal, status.TasksTotal)
	}
	if unmarshaled.TasksDone != status.TasksDone {
		t.Errorf("TasksDone: got %v, want %v", unmarshaled.TasksDone, status.TasksDone)
	}
	if unmarshaled.LocalChanges != status.LocalChanges {
		t.Errorf("LocalChanges: got %v, want %v", unmarshaled.LocalChanges, status.LocalChanges)
	}
	if unmarshaled.RemoteChanges != status.RemoteChanges {
		t.Errorf("RemoteChanges: got %v, want %v", unmarshaled.RemoteChanges, status.RemoteChanges)
	}
	if unmarshaled.LastPullAt == nil || !unmarshaled.LastPullAt.Equal(now) {
		t.Errorf("LastPullAt: got %v, want %v", unmarshaled.LastPullAt, now)
	}
	if unmarshaled.LastPushAt == nil || !unmarshaled.LastPushAt.Equal(now) {
		t.Errorf("LastPushAt: got %v, want %v", unmarshaled.LastPushAt, now)
	}
}
