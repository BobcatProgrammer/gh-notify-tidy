package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show notification statistics",
	Long: `Display a per-repository breakdown of your notifications along with
subscription suggestions to help reduce noise.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := gh.NewClient(flagHost)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		filter := gh.Filter{
			Repo: flagRepo,
			Org:  flagOrg,
		}

		threads, err := client.ListAll(filter)
		if err != nil {
			return err
		}

		if len(threads) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No notifications found.")
			return nil
		}

		stats := gh.ComputeStats(threads)

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "REPOSITORY\tTOTAL\tUNREAD\tTOP REASON\tSUGGESTION")
		_, _ = fmt.Fprintln(w, "----------\t-----\t------\t----------\t----------")

		for _, s := range stats {
			topReason := topReasonKey(s.ByReason)
			_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\n",
				s.Repo, s.Total, s.Unread, topReason, s.Suggestion)
		}

		return w.Flush()
	},
}

func topReasonKey(m map[string]int) string {
	top, count := "", 0

	for k, v := range m {
		if v > count {
			top, count = k, v
		}
	}

	return top
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
