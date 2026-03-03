package commands

import (
	"testing"

	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/internal/testutil"
)

func TestRemoteRemove_Force(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "origin"
	if err := config.SaveRemoteConfig(name, "https://x.com", "token", "."); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	if !config.RemoteConfigExists(name) {
		t.Fatal("remote should exist before remove")
	}

	rc := NewRootCommand(config.DefaultConfig(), nil)
	rc.SetArgs([]string{"remote", "remove", name, "--force"})
	rc.SilenceUsage()

	if err := rc.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if config.RemoteConfigExists(name) {
		t.Error("remote should not exist after remove")
	}
}

func TestRemoteRemove_Nonexistent(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	rc := NewRootCommand(config.DefaultConfig(), nil)
	rc.SetArgs([]string{"remote", "remove", "nonexistent", "--force"})
	rc.SilenceUsage()

	err := rc.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent remote")
	}
	if err.Error() != `remote "nonexistent" does not exist` {
		t.Errorf("unexpected error: %v", err)
	}
}
