package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	TasksDirectory string
	DefaultRemote  string
	Editor         string
}

const (
	DefaultTasksDir = "./tasks"
	DefaultRemote   = "origin"
	DefaultEditor   = "vim"
)

func DefaultConfig() *Config {
	return &Config{
		TasksDirectory: DefaultTasksDir,
		DefaultRemote:  DefaultRemote,
		Editor:         DefaultEditor,
	}
}

func Load() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}
	if err := ensureDir(configDir); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	return DefaultConfig(), nil
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, 0755)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".tasklean"), nil
}

func GetTasksDir() (string, error) {
	cfg, err := Load()
	if err != nil {
		return DefaultTasksDir, nil
	}
	return cfg.TasksDirectory, nil
}
