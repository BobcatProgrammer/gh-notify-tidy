package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

var (
	allOlderThan   int
	allActionFlags actionFlags
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Find all stale notifications (PR/issue staleness + age)",
	Long: `Combines PR/issue staleness detection and age-based filtering to find
notifications that are unlikely to need further action.

Includes:
  - Pull requests that have been merged or closed
  - Pull requests already approved by a colleague
  - Issues that have been closed
  - Notifications not updated in --older-than days (default: 0, disabled)

By default the matching notifications are printed. Use --done, --read, or
--mute to act on them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := allActionFlags.validate(); err != nil {
			return err
		}

		client, err := gh.NewClient(flagHost)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		filter := gh.Filter{
			Repo: flagRepo,
			Org:  flagOrg,
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Fetching notifications…")

		threads, err := client.ListAll(filter)
		if err != nil {
			return err
		}

		viewerLogin, err := client.GetAuthenticatedUser()
		if err != nil {
			return fmt.Errorf("get authenticated user: %w", err)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Checking %d notification(s) for staleness…\n", len(threads))

		staleThreads := gh.StaleOnly(gh.CheckStalePR(client, threads, viewerLogin))

		// Collect stale thread IDs to avoid duplicates when also adding old ones.
		staleIDs := make(map[string]bool, len(staleThreads))
		flat := make([]gh.Thread, len(staleThreads))

		for i, st := range staleThreads {
			flat[i] = st.Thread
			staleIDs[st.Thread.ID] = true
		}

		// Add old threads that weren't already caught by staleness checks.
		if allOlderThan > 0 {
			cutoff := time.Now().Add(-time.Duration(allOlderThan) * 24 * time.Hour)

			for _, t := range threads {
				if !staleIDs[t.ID] && t.UpdatedAt.Before(cutoff) {
					flat = append(flat, t)
				}
			}
		}

		if len(flat) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No stale notifications found.")
			return nil
		}

		// Print stale PR/issue groups.
		if len(staleThreads) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nStale PR/issue notifications (%d):\n", len(staleThreads))

			for _, g := range gh.GroupByRepoReason(staleThreads) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  %s  ·  %s  (%d)\n",
					g.Repo, string(g.StaleReason), len(g.Threads))

				for _, t := range g.Threads {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", t.Subject.Title)
				}
			}
		}

		// Print old notifications that are not already in the stale list.
		oldCount := len(flat) - len(staleThreads)
		if oldCount > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"\nNotifications older than %d day(s) (%d):\n", allOlderThan, oldCount)

			for _, t := range flat[len(staleThreads):] {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s / %s\n",
					t.Repository.FullName, t.Subject.Title)
			}
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout())

		return allActionFlags.apply(cmd, client, flat, flagDryRun)
	},
}

func init() {
	allCmd.Flags().IntVar(&allOlderThan, "older-than", 0,
		"also include notifications not updated in this many days (0 = disabled)")
	allActionFlags.register(allCmd)
	rootCmd.AddCommand(allCmd)
}
