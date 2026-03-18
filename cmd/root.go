package cmd

import (
	"github.com/spf13/cobra"
)

// Global flag values shared across subcommands.
var (
	flagRepo   string
	flagOrg    string
	flagHost   string
	flagDryRun bool
)

// rootCmd is the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "notify-tidy",
	Short: "Clean up and manage GitHub notifications",
	Long: `gh notify-tidy helps you triage, bulk-archive, mute, and understand
your GitHub notifications.

Use one of the subcommands to get started:

  gh notify-tidy stats        — show notification statistics
  gh notify-tidy interactive  — guided interactive cleanup (recommended)
  gh notify-tidy read         — mark notifications as read
  gh notify-tidy done         — archive / unsubscribe notifications
  gh notify-tidy mute         — mute notification threads`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagRepo, "repo", "", "filter to a single repo (owner/repo)")
	rootCmd.PersistentFlags().StringVar(&flagOrg, "org", "", "filter to an organisation")
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", "", "GitHub hostname (for GitHub Enterprise Server)")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "print what would be done without making changes")
}
