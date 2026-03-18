package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
	"github.com/BobcatProgrammer/gh-notify-tidy/internal/tui"
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i"},
	Short:   "Interactive guided notification cleanup (TUI)",
	Long: `Walk through your stale notifications in a Bubble Tea TUI.

The tool fetches all notifications, checks which ones no longer require
action (merged/closed PRs, approved PRs, closed issues), then presents
them grouped by repository and reason for you to bulk-triage.

Steps:
  1. Loading + staleness check
  2. Statistics overview
  3. Stale notification groups — mark each group as done/read/mute/skip
  4. Confirm — review and apply selected actions

Keys: r=read  d=done  m=mute  s=skip  enter/n=next step  q=quit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := gh.NewClient(flagHost)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		filter := gh.Filter{
			Repo: flagRepo,
			Org:  flagOrg,
		}

		model := tui.New(client, filter, flagDryRun)

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("interactive TUI: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}
