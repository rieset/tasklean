package tracker

import (
	"context"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
	"github.com/rie/tasklean/pkg/tracker/plane"
)

// ResolvingTracker delegates to Plane or Stub based on remote config.
// Uses Plane when config has Workspace and Project set.
type ResolvingTracker struct {
	plane *plane.PlaneTracker
	stub  *StubTracker
}

// NewResolvingTracker creates a tracker that auto-selects Plane or Stub.
func NewResolvingTracker() *ResolvingTracker {
	return &ResolvingTracker{
		plane: plane.NewPlaneTracker(),
		stub:  &StubTracker{},
	}
}

// FetchTasks delegates to Plane or Stub based on cfg.Workspace and cfg.Project.
func (r *ResolvingTracker) FetchTasks(ctx context.Context, cfg *config.RemoteConfig, opts *FetchOptions) ([]*models.Task, error) {
	if cfg.Workspace != "" && cfg.Project != "" {
		po := &plane.FetchOptions{}
		if opts != nil {
			po.From = opts.From
			po.Assignee = opts.Assignee
			po.Debug = opts.Debug
			po.SkipModules = opts.SkipModules
		}
		return r.plane.FetchTasks(ctx, cfg, po)
	}
	return r.stub.FetchTasks(ctx, cfg, opts)
}

// UpdateTask delegates to Plane or Stub.
func (r *ResolvingTracker) UpdateTask(ctx context.Context, cfg *config.RemoteConfig, task *models.Task) error {
	if cfg.Workspace != "" && cfg.Project != "" {
		return r.plane.UpdateTask(ctx, cfg, task)
	}
	return r.stub.UpdateTask(ctx, cfg, task)
}

// CreateTask delegates to Plane or Stub.
func (r *ResolvingTracker) CreateTask(ctx context.Context, cfg *config.RemoteConfig, task *models.Task) (*models.Task, error) {
	if cfg.Workspace != "" && cfg.Project != "" {
		return r.plane.CreateTask(ctx, cfg, task)
	}
	return r.stub.CreateTask(ctx, cfg, task)
}

// SyncTaskModule delegates to Plane or Stub.
func (r *ResolvingTracker) SyncTaskModule(ctx context.Context, cfg *config.RemoteConfig, taskID, fromModule, toModule string) error {
	if cfg.Workspace != "" && cfg.Project != "" {
		return r.plane.SyncTaskModule(ctx, cfg, taskID, fromModule, toModule)
	}
	return r.stub.SyncTaskModule(ctx, cfg, taskID, fromModule, toModule)
}
