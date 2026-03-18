package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

var muteOlderThan int

var muteCmd = &cobra.Command{
	Use:   "mute",
	Short: "Mute notification threads",
	Long: `Set ignored=true on the subscription for matching notification threads.

Muted threads will no longer generate notifications. Use --older-than to
restrict to notifications that have not been updated in N days.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := gh.NewClient(flagHost)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		filter := gh.Filter{
			Repo: flagRepo,
			Org:  flagOrg,
		}
		if muteOlderThan > 0 {
			filter.OlderThan = time.Duration(muteOlderThan) * 24 * time.Hour
		}

		threads, err := client.ListAll(filter)
		if err != nil {
			return err
		}

		if len(threads) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No matching notifications found.")
			return nil
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Muting %d notification thread(s)", len(threads))
		if flagDryRun {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), " (dry run)")
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "…")

		var errs int

		for _, t := range threads {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  mute  %s / %s\n",
				t.Repository.FullName, t.Subject.Title)

			if flagDryRun {
				continue
			}

			if err := client.Mute(t.ID); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  error: %v\n", err)
				errs++
			}
		}

		if errs > 0 {
			return fmt.Errorf("%d error(s) occurred", errs)
		}

		return nil
	},
}

func init() {
	muteCmd.Flags().IntVar(&muteOlderThan, "older-than", 0,
		"only include notifications not updated in N days")
	rootCmd.AddCommand(muteCmd)
}
