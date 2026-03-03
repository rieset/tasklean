package tracker

import (
	"context"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
)

// FetchOptions holds optional filters for FetchTasks.
type FetchOptions struct {
	From        string // date filter (YYYY-MM-DD), tasks updated from this date
	Assignee    string // email of assignee, filter tasks assigned to this user
	Debug       bool   // dump raw API response for first issue
	SkipModules bool   // skip module resolution (faster, use when only IDs are needed)
}

// Tracker fetches and pushes tasks to a remote task tracker.
type Tracker interface {
	FetchTasks(ctx context.Context, cfg *config.RemoteConfig, opts *FetchOptions) ([]*models.Task, error)
	UpdateTask(ctx context.Context, cfg *config.RemoteConfig, task *models.Task) error
	CreateTask(ctx context.Context, cfg *config.RemoteConfig, task *models.Task) (*models.Task, error)
	// SyncTaskModule moves a task from one module to another in the tracker.
	// fromModule is the current module in the tracker (empty if none),
	// toModule is the desired module (empty to remove from module without adding to another).
	SyncTaskModule(ctx context.Context, cfg *config.RemoteConfig, taskID, fromModule, toModule string) error
}
