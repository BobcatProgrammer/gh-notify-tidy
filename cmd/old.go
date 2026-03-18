package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

var (
	oldDays        int
	oldActionFlags actionFlags
)

var oldCmd = &cobra.Command{
	Use:   "old",
	Short: "Find notifications older than N days",
	Long: `List notifications that have not been updated in N days (default: 30).
These are often safe to archive once the work they relate to is complete.

By default the matching notifications are printed. Use --done, --read, or
--mute to act on them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := oldActionFlags.validate(); err != nil {
			return err
		}

		client, err := gh.NewClient(flagHost)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		filter := gh.Filter{
			Repo:      flagRepo,
			Org:       flagOrg,
			OlderThan: time.Duration(oldDays) * 24 * time.Hour,
		}

		threads, err := client.ListAll(filter)
		if err != nil {
			return err
		}

		if len(threads) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"No notifications older than %d day(s) found.\n", oldDays)
			return nil
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(),
			"Found %d notification(s) older than %d day(s):\n\n", len(threads), oldDays)

		// Group by repo for readable output.
		byRepo := make(map[string][]gh.Thread)
		repoOrder := []string{}

		for _, t := range threads {
			repo := t.Repository.FullName
			if _, ok := byRepo[repo]; !ok {
				repoOrder = append(repoOrder, repo)
			}

			byRepo[repo] = append(byRepo[repo], t)
		}

		for _, repo := range repoOrder {
			ts := byRepo[repo]
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s  (%d)\n", repo, len(ts))

			for _, t := range ts {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    [%s] %s\n",
					t.Subject.Type, t.Subject.Title)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}

		return oldActionFlags.apply(cmd, client, threads, flagDryRun)
	},
}

func init() {
	oldCmd.Flags().IntVar(&oldDays, "older-than", 30,
		"include notifications not updated in this many days")
	oldActionFlags.register(oldCmd)
	rootCmd.AddCommand(oldCmd)
}
