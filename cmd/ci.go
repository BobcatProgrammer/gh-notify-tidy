package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Find stale CI activity notifications (not yet implemented)",
	Long: `Identify ci_activity notifications where the failing check run has
since been fixed by a successful run on the same pull request.

This feature is not yet implemented. Track progress:
https://github.com/BobcatProgrammer/gh-notify-tidy/issues/12`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(),
			"CI staleness detection is not yet implemented.")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(),
			"Track progress: https://github.com/BobcatProgrammer/gh-notify-tidy/issues/12")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(ciCmd)
}
