package testutil

import (
	"os"
	"testing"
)

func SetTestHome(t *testing.T, dir string) {
	t.Helper()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", dir); err != nil {
		t.Fatalf("setenv HOME: %v", err)
	}
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
	})
}
