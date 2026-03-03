package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rie/tasklean/internal/testutil"
)

func TestGetConfigDir(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	configDir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir: %v", err)
	}
	want := filepath.Join(tmp, ".tasklean")
	if configDir != want {
		t.Errorf("GetConfigDir() = %q, want %q", configDir, want)
	}
}

func TestLoadCreatesConfigDir(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	configDir := filepath.Join(tmp, ".tasklean")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("config dir %q was not created", configDir)
	}
}

func TestSaveAndLoadRemoteConfig(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := Load(); err != nil {
		t.Fatalf("Load (ensure dir): %v", err)
	}

	name := "origin"
	url := "https://task.example.com"
	token := "secret-token"
	directory := "./tasks"

	if err := SaveRemoteConfig(name, url, token, directory); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	cfgPath := filepath.Join(tmp, ".tasklean", name+".conf")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Errorf("config file %q was not created", cfgPath)
	}

	loaded, err := LoadRemoteConfig(name)
	if err != nil {
		t.Fatalf("LoadRemoteConfig: %v", err)
	}
	if loaded.Name != name {
		t.Errorf("Name = %q, want %q", loaded.Name, name)
	}
	if loaded.URL != url {
		t.Errorf("URL = %q, want %q", loaded.URL, url)
	}
	if loaded.Token != token {
		t.Errorf("Token = %q, want %q", loaded.Token, token)
	}
	if loaded.Directory != directory {
		t.Errorf("Directory = %q, want %q", loaded.Directory, directory)
	}
}

func TestLoadRemoteConfigNotFound(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	_, err := LoadRemoteConfig("nonexistent")
	if err == nil {
		t.Error("LoadRemoteConfig expected error for nonexistent remote")
	}
}

func TestRemoveRemoteConfig(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "test-remote"
	if err := SaveRemoteConfig(name, "https://x.com", "token", "."); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	if !RemoteConfigExists(name) {
		t.Error("RemoteConfigExists should be true after save")
	}

	if err := RemoveRemoteConfig(name); err != nil {
		t.Fatalf("RemoveRemoteConfig: %v", err)
	}

	if RemoteConfigExists(name) {
		t.Error("RemoteConfigExists should be false after remove")
	}

	_, err := LoadRemoteConfig(name)
	if err == nil {
		t.Error("LoadRemoteConfig expected error after remove")
	}
}

func TestListRemoteConfigs(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	configs, err := ListRemoteConfigs()
	if err != nil {
		t.Fatalf("ListRemoteConfigs: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("ListRemoteConfigs empty dir: got %d configs", len(configs))
	}

	if err := SaveRemoteConfig("a", "https://a.com", "t1", "."); err != nil {
		t.Fatalf("SaveRemoteConfig a: %v", err)
	}
	if err := SaveRemoteConfig("b", "https://b.com", "t2", "./b"); err != nil {
		t.Fatalf("SaveRemoteConfig b: %v", err)
	}

	configs, err = ListRemoteConfigs()
	if err != nil {
		t.Fatalf("ListRemoteConfigs: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("ListRemoteConfigs: got %d, want 2", len(configs))
	}
}

func TestRemoveRemoteConfig_InvalidName(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	err := RemoveRemoteConfig("../../etc")
	if err == nil {
		t.Error("expected error for invalid name")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid remote name") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateRemoteLastPullAt(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetTestHome(t, tmp)

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	name := "origin"
	if err := SaveRemoteConfig(name, "https://x.com", "token", "."); err != nil {
		t.Fatalf("SaveRemoteConfig: %v", err)
	}

	timestamp := "2024-06-15T12:00:00Z"
	if err := UpdateRemoteLastPullAt(name, timestamp); err != nil {
		t.Fatalf("UpdateRemoteLastPullAt: %v", err)
	}

	cfg, err := LoadRemoteConfig(name)
	if err != nil {
		t.Fatalf("LoadRemoteConfig: %v", err)
	}
	if cfg.LastPullAt != timestamp {
		t.Errorf("LastPullAt = %q, want %q", cfg.LastPullAt, timestamp)
	}
}

func TestParsePlaneIssuesURL(t *testing.T) {
	tests := []struct {
		url       string
		base      string
		workspace string
		project   string
		ok        bool
	}{
		{
			"https://plane.example.com/my-workspace/projects/3481d8a2-bcdc-4553-aa8c-922bcf009255/issues/",
			"https://plane.example.com",
			"my-workspace",
			"3481d8a2-bcdc-4553-aa8c-922bcf009255",
			true,
		},
		{
			"https://app.plane.so/workspace/projects/uuid-here/issues",
			"https://app.plane.so",
			"workspace",
			"uuid-here",
			true,
		},
		{"https://example.com", "", "", "", false},
		{"", "", "", "", false},
	}
	for _, tt := range tests {
		base, ws, prj, ok := ParsePlaneIssuesURL(tt.url)
		if ok != tt.ok || base != tt.base || ws != tt.workspace || prj != tt.project {
			t.Errorf("ParsePlaneIssuesURL(%q) = (%q, %q, %q, %v), want (%q, %q, %q, %v)",
				tt.url, base, ws, prj, ok, tt.base, tt.workspace, tt.project, tt.ok)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.TasksDirectory != DefaultTasksDir {
		t.Errorf("TasksDirectory = %q, want %q", cfg.TasksDirectory, DefaultTasksDir)
	}
	if cfg.DefaultRemote != DefaultRemote {
		t.Errorf("DefaultRemote = %q, want %q", cfg.DefaultRemote, DefaultRemote)
	}
	if cfg.Editor != DefaultEditor {
		t.Errorf("Editor = %q, want %q", cfg.Editor, DefaultEditor)
	}
}
