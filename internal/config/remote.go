package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ParsePlaneIssuesURL parses a Plane project issues URL.
// Example: https://plane.example.com/my-workspace/projects/3481d8a2-bcdc-4553-aa8c-922bcf009255/issues/
// Returns: baseURL, workspace, projectID, ok
func ParsePlaneIssuesURL(raw string) (baseURL, workspace, projectID string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", false
	}
	// Pattern: /{workspace}/projects/{project_id}/... (issues, work-items, or nothing)
	re := regexp.MustCompile(`^/([^/]+)/projects/([^/]+)/`)
	m := re.FindStringSubmatch(u.Path)
	if m == nil {
		return "", "", "", false
	}
	workspace = m[1]
	projectID = m[2]
	baseURL = u.Scheme + "://" + u.Host
	baseURL = strings.TrimSuffix(baseURL, "/")
	return baseURL, workspace, projectID, true
}

func validateRemoteName(name string) error {
	if name == "" {
		return fmt.Errorf("remote name cannot be empty")
	}
	if strings.Contains(name, string(os.PathSeparator)) || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid remote name: %q", name)
	}
	return nil
}

type RemoteConfig struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Token      string `json:"token"`
	Directory  string `json:"directory"`
	Workspace  string `json:"workspace,omitempty"` // Plane: workspace slug
	Project    string `json:"project,omitempty"`   // Plane: project ID (UUID)
	Cloud      bool   `json:"cloud,omitempty"`     // Plane: use cloud API (work-items endpoint)
	CreatedAt  string `json:"created_at"`
	LastPullAt string `json:"last_pull_at,omitempty"`
	LastPushAt string `json:"last_push_at,omitempty"`
}

func SaveRemoteConfig(name, url, token, directory string) error {
	return SaveRemoteConfigPlane(name, url, token, directory, "", "", false)
}

func SaveRemoteConfigPlane(name, url, token, directory, workspace, project string, cloud bool) error {
	if err := validateRemoteName(name); err != nil {
		return err
	}
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}

	if err := ensureDir(configDir); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	configPath := filepath.Join(configDir, name+".conf")

	remoteCfg := RemoteConfig{
		Name:      name,
		URL:       url,
		Token:     token,
		Directory: directory,
		Workspace: workspace,
		Project:   project,
		Cloud:     cloud,
		CreatedAt: "2024-01-01T00:00:00Z",
	}

	data, err := json.MarshalIndent(remoteCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func LoadRemoteConfig(name string) (*RemoteConfig, error) {
	if err := validateRemoteName(name); err != nil {
		return nil, err
	}
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	configPath := filepath.Join(configDir, name+".conf")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var remoteCfg RemoteConfig
	if err := json.Unmarshal(data, &remoteCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &remoteCfg, nil
}

func RemoveRemoteConfig(name string) error {
	if err := validateRemoteName(name); err != nil {
		return err
	}
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}

	configPath := filepath.Join(configDir, name+".conf")

	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to remove config: %w", err)
	}

	return nil
}

func ListRemoteConfigs() ([]RemoteConfig, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RemoteConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read config dir: %w", err)
	}

	var configs []RemoteConfig
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".conf" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-len(".conf")]
		cfg, err := LoadRemoteConfig(name)
		if err != nil {
			continue
		}
		configs = append(configs, *cfg)
	}

	return configs, nil
}

func RemoteConfigExists(name string) bool {
	_, err := LoadRemoteConfig(name)
	return err == nil
}

func UpdateRemoteLastPullAt(name, timestamp string) error {
	cfg, err := LoadRemoteConfig(name)
	if err != nil {
		return err
	}
	cfg.LastPullAt = timestamp
	return saveRemoteConfig(cfg)
}

func UpdateRemoteLastPushAt(name, timestamp string) error {
	cfg, err := LoadRemoteConfig(name)
	if err != nil {
		return err
	}
	cfg.LastPushAt = timestamp
	return saveRemoteConfig(cfg)
}

func SaveRemoteConfigFromStruct(cfg *RemoteConfig) error {
	return saveRemoteConfig(cfg)
}

func saveRemoteConfig(cfg *RemoteConfig) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config dir: %w", err)
	}
	configPath := filepath.Join(configDir, cfg.Name+".conf")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}
