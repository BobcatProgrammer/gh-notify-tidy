package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
	"github.com/BobcatProgrammer/gh-notify-tidy/internal/tui"
)

var interactiveOldDays int

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i"},
	Short:   "Interactive guided notification cleanup (TUI)",
	Long: `Walk through your notifications step-by-step in a Bubble Tea TUI.

Steps:
  1. Statistics — per-repo breakdown with suggestions
  2. Old notifications — triage notifications older than N days
  3. Closed/merged PR notifications — triage read PR threads
  4. Already-read notifications — triage all other read threads
  5. Confirm — review and apply selected actions

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

		model := tui.New(client, filter, flagDryRun, interactiveOldDays)

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("interactive TUI: %w", err)
		}

		return nil
	},
}

func init() {
	interactiveCmd.Flags().IntVar(&interactiveOldDays, "old-days", 30,
		"notifications older than this many days are shown in the 'old' step")
	rootCmd.AddCommand(interactiveCmd)
}
