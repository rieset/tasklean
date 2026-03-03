package main

import (
	"os"

	"github.com/rie/tasklean/internal/commands"
	"github.com/rie/tasklean/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	rootCmd := commands.NewRootCommand(cfg, nil)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
