package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

var prActionFlags actionFlags

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Find stale PR and issue notifications",
	Long: `Identify notifications that no longer require action because the
underlying pull request or issue has been closed, merged, or already
reviewed/approved by a colleague.

By default the matching notifications are printed. Use --done, --read, or
--mute to act on them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := prActionFlags.validate(); err != nil {
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

		stale := gh.StaleOnly(gh.CheckStalePR(client, threads, viewerLogin))

		if len(stale) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No stale PR/issue notifications found.")
			return nil
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nFound %d stale notification(s):\n", len(stale))

		for _, g := range gh.GroupByRepoReason(stale) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  %s  ·  %s  (%d)\n",
				g.Repo, string(g.StaleReason), len(g.Threads))

			for _, t := range g.Threads {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", t.Subject.Title)
			}
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout())

		flat := make([]gh.Thread, len(stale))
		for i, st := range stale {
			flat[i] = st.Thread
		}

		return prActionFlags.apply(cmd, client, flat, flagDryRun)
	},
}

func init() {
	prActionFlags.register(prCmd)
	rootCmd.AddCommand(prCmd)
}
