package models

import (
	"strings"
	"time"
)

type TaskStatus string

const (
	StatusBacklog    TaskStatus = "backlog"
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusInReview   TaskStatus = "in_review"
	StatusDone       TaskStatus = "done"
	StatusCancelled  TaskStatus = "cancelled"
)

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      TaskStatus `json:"status"`
	Module      string     `json:"module,omitempty"`
	Assignees   []string   `json:"assignees,omitempty"` // emails or display names for folder structure
	Remote      string     `json:"remote,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty"`
}

func (s TaskStatus) String() string {
	return string(s)
}

// DisplayName returns human-readable status name for file header.
func (s TaskStatus) DisplayName() string {
	switch s {
	case StatusBacklog:
		return "Backlog"
	case StatusTodo:
		return "Todo"
	case StatusInProgress:
		return "In Progress"
	case StatusInReview:
		return "In Review"
	case StatusDone:
		return "Done"
	case StatusCancelled:
		return "Cancelled"
	default:
		return string(s)
	}
}

func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusBacklog, StatusTodo, StatusInProgress, StatusInReview, StatusDone, StatusCancelled:
		return true
	}
	return false
}

// StatusToEmoji returns emoji for status (for display and push).
func (s TaskStatus) Emoji() string {
	switch s {
	case StatusBacklog:
		return "📥"
	case StatusTodo:
		return "📋"
	case StatusInProgress:
		return "🚀"
	case StatusInReview:
		return "👀"
	case StatusDone:
		return "✅"
	case StatusCancelled:
		return "❌"
	default:
		return "📋"
	}
}

// EmojiToStatus maps emoji to status. Returns empty string if unknown.
func EmojiToStatus(emoji string) TaskStatus {
	switch strings.TrimSpace(emoji) {
	case "📥":
		return StatusBacklog
	case "📋", "📝":
		return StatusTodo
	case "🚀", "🔄", "⏳":
		return StatusInProgress
	case "👀", "🔍":
		return StatusInReview
	case "✅", "✔️", "✔":
		return StatusDone
	case "❌", "🚫":
		return StatusCancelled
	default:
		return ""
	}
}
