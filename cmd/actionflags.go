package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

// actionFlags holds the mutually exclusive --done / --read / --mute flags
// used by the pr, all, and old commands.
type actionFlags struct {
	done bool
	read bool
	mute bool
}

// register adds --done, --read, and --mute to cmd.
func (f *actionFlags) register(c *cobra.Command) {
	c.Flags().BoolVar(&f.done, "done", false, "archive (delete subscription) matching notifications")
	c.Flags().BoolVar(&f.read, "read", false, "mark matching notifications as read")
	c.Flags().BoolVar(&f.mute, "mute", false, "mute matching notification threads")
}

// validate returns an error if more than one action flag is set.
func (f *actionFlags) validate() error {
	n := 0
	for _, v := range []bool{f.done, f.read, f.mute} {
		if v {
			n++
		}
	}

	if n > 1 {
		return fmt.Errorf("--done, --read, and --mute are mutually exclusive")
	}

	return nil
}

// apply executes the chosen action on all provided threads.
// It prints each thread as it is processed and respects --dry-run.
func (f *actionFlags) apply(cmd *cobra.Command, client *gh.Client, threads []gh.Thread, dryRun bool) error {
	if !f.done && !f.read && !f.mute {
		return nil // nothing to do
	}

	verb := "read"
	switch {
	case f.done:
		verb = "done"
	case f.mute:
		verb = "mute"
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Applying '%s' to %d notification(s)", verb, len(threads))
	if dryRun {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), " (dry run)")
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "…")

	var errs int

	for _, t := range threads {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s / %s\n",
			verb, t.Repository.FullName, t.Subject.Title)

		if dryRun {
			continue
		}

		var err error

		switch {
		case f.done:
			err = client.Done(t.ID)
		case f.read:
			err = client.MarkRead(t.ID)
		case f.mute:
			err = client.Mute(t.ID)
		}

		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  error: %v\n", err)
			errs++
		}
	}

	if errs > 0 {
		return fmt.Errorf("%d error(s) occurred", errs)
	}

	return nil
}
