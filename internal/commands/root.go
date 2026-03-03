package commands

import (
	"github.com/rie/tasklean/internal/config"
	"github.com/rie/tasklean/pkg/tracker"
	"github.com/spf13/cobra"
)

type RootCommand struct {
	cmd     *cobra.Command
	cfg     *config.Config
	tracker tracker.Tracker
}

func NewRootCommand(cfg *config.Config, tr tracker.Tracker) *RootCommand {
	if tr == nil {
		tr = tracker.NewResolvingTracker()
	}
	rc := &RootCommand{
		cfg:     cfg,
		tracker: tr,
	}

	rootCmd := &cobra.Command{
		Use:   "tasklean",
		Short: "Task tracker CLI with file-based workflow",
		Long: `Tasklean - CLI tool for managing tasks from task trackers as local files.
Built with Bubble Tea for interactive TUI experience.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand(
		newRemoteCommand(),
		newPullCommand(rc.tracker),
		newPushCommand(rc.tracker),
		newStatusCommand(),
		newResolveCommand(),
	)

	rc.cmd = rootCmd
	return rc
}

func (rc *RootCommand) Execute() error {
	return rc.cmd.Execute()
}

func (rc *RootCommand) SetArgs(args []string) {
	rc.cmd.SetArgs(args)
}

func (rc *RootCommand) SilenceUsage() {
	rc.cmd.SilenceUsage = true
	rc.cmd.SilenceErrors = true
}
