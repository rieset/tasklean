package tracker

import (
	"context"
	"errors"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/models"
)

// ErrNotImplemented is returned when the tracker API is not yet implemented.
var ErrNotImplemented = errors.New("tracker API not implemented")

// StubTracker returns ErrNotImplemented for all requests.
// Used until a real tracker (Jira, Linear, etc.) is implemented.
type StubTracker struct{}

// FetchTasks returns ErrNotImplemented.
func (s *StubTracker) FetchTasks(_ context.Context, _ *config.RemoteConfig, _ *FetchOptions) ([]*models.Task, error) {
	return nil, ErrNotImplemented
}

// UpdateTask returns ErrNotImplemented.
func (s *StubTracker) UpdateTask(_ context.Context, _ *config.RemoteConfig, _ *models.Task) error {
	return ErrNotImplemented
}

// CreateTask returns ErrNotImplemented.
func (s *StubTracker) CreateTask(_ context.Context, _ *config.RemoteConfig, _ *models.Task) (*models.Task, error) {
	return nil, ErrNotImplemented
}

// SyncTaskModule returns ErrNotImplemented.
func (s *StubTracker) SyncTaskModule(_ context.Context, _ *config.RemoteConfig, _, _, _ string) error {
	return ErrNotImplemented
}
