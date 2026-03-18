package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

var (
	doneOlderThan int
	doneOnlyRead  bool
	doneClosedPRs bool
	doneAuto      bool
)

var doneCmd = &cobra.Command{
	Use:   "done",
	Short: "Archive notifications (delete subscription)",
	Long: `Delete the subscription for matching notifications ("Done" / archive).

Use --older-than, --read, and --closed-prs to narrow which notifications are
processed.

Use --auto to automatically detect and archive all stale notifications
(merged/closed PRs, approved PRs, closed issues). This is equivalent to
running 'gh notify-tidy all --done'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if doneAuto {
			return runDoneAuto(cmd)
		}

		return runDoneManual(cmd)
	},
}

func runDoneAuto(cmd *cobra.Command) error {
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
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No stale notifications found.")
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nFound %d stale notification(s):\n", len(stale))

	for _, st := range stale {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  [%s]  %s / %s\n",
			string(st.StaleReason), st.Thread.Repository.FullName, st.Thread.Subject.Title)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Archiving %d notification(s)", len(stale))

	if flagDryRun {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), " (dry run)")
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "…")

	var errs int

	for _, st := range stale {
		t := st.Thread
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  done  %s / %s\n",
			t.Repository.FullName, t.Subject.Title)

		if flagDryRun {
			continue
		}

		if err := client.Done(t.ID); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  error: %v\n", err)
			errs++
		}
	}

	if errs > 0 {
		return fmt.Errorf("%d error(s) occurred", errs)
	}

	return nil
}

func runDoneManual(cmd *cobra.Command) error {
	client, err := gh.NewClient(flagHost)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	filter := gh.Filter{
		Repo:     flagRepo,
		Org:      flagOrg,
		OnlyRead: doneOnlyRead,
	}
	if doneOlderThan > 0 {
		filter.OlderThan = time.Duration(doneOlderThan) * 24 * time.Hour
	}

	threads, err := client.ListAll(filter)
	if err != nil {
		return err
	}

	if doneClosedPRs {
		threads = filterClosedPRThreads(threads)
	}

	if len(threads) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No matching notifications found.")
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Archiving %d notification(s)", len(threads))
	if flagDryRun {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), " (dry run)")
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "…")

	var errs int

	for _, t := range threads {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  done  %s / %s\n",
			t.Repository.FullName, t.Subject.Title)

		if flagDryRun {
			continue
		}

		if err := client.Done(t.ID); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  error: %v\n", err)
			errs++
		}
	}

	if errs > 0 {
		return fmt.Errorf("%d error(s) occurred", errs)
	}

	return nil
}

// filterClosedPRThreads returns only threads that are for PullRequests and
// whose PR is closed or merged.  Because a real closed-PR check requires an
// API call per thread we use the cheap heuristic: the thread is for a PR and
// is already read (GitHub marks PR threads read when the PR is merged/closed
// in many cases).  Callers wanting the full check should use the interactive
// command which calls GetPR per thread.
func filterClosedPRThreads(threads []gh.Thread) []gh.Thread {
	var out []gh.Thread

	for _, t := range threads {
		if t.Subject.Type == "PullRequest" && !t.Unread {
			out = append(out, t)
		}
	}

	return out
}

func init() {
	doneCmd.Flags().IntVar(&doneOlderThan, "older-than", 0,
		"only include notifications not updated in N days")
	doneCmd.Flags().BoolVar(&doneOnlyRead, "read", false,
		"only include already-read notifications")
	doneCmd.Flags().BoolVar(&doneClosedPRs, "closed-prs", false,
		"only include read notifications for closed/merged PRs (heuristic)")
	doneCmd.Flags().BoolVar(&doneAuto, "auto", false,
		"automatically detect and archive all stale notifications")
	rootCmd.AddCommand(doneCmd)
}
