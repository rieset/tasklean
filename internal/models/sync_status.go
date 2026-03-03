package models

import "time"

type SyncDirection string

const (
	SyncDirectionPull SyncDirection = "pull"
	SyncDirectionPush SyncDirection = "push"
	SyncDirectionBoth SyncDirection = "both"
)

type SyncStatus struct {
	RemoteName    string        `json:"remote_name"`
	Direction     SyncDirection `json:"direction"`
	LastPullAt    *time.Time    `json:"last_pull_at,omitempty"`
	LastPushAt    *time.Time    `json:"last_push_at,omitempty"`
	TasksTotal    int           `json:"tasks_total"`
	TasksDone     int           `json:"tasks_done"`
	LocalChanges  int           `json:"local_changes"`
	RemoteChanges int           `json:"remote_changes"`
	Error         string        `json:"error,omitempty"`
}

func NewSyncStatus(remoteName string) *SyncStatus {
	return &SyncStatus{
		RemoteName: remoteName,
		Direction:  SyncDirectionBoth,
	}
}

func (s *SyncStatus) MarkPulled() {
	now := time.Now()
	s.LastPullAt = &now
}

func (s *SyncStatus) MarkPushed() {
	now := time.Now()
	s.LastPushAt = &now
}
