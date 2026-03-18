package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colours
	colorPrimary  = lipgloss.Color("69")  // blue
	colorSuccess  = lipgloss.Color("76")  // green
	colorWarning  = lipgloss.Color("214") // orange
	colorDanger   = lipgloss.Color("196") // red
	colorMuted    = lipgloss.Color("240") // grey
	colorSelected = lipgloss.Color("212") // pink

	// Base text styles
	Bold    = lipgloss.NewStyle().Bold(true)
	Muted   = lipgloss.NewStyle().Foreground(colorMuted)
	Primary = lipgloss.NewStyle().Foreground(colorPrimary)

	// Status badges
	BadgeRead   = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	BadgeDone   = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	BadgeMute   = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	BadgeSkip   = lipgloss.NewStyle().Foreground(colorMuted)
	BadgeDanger = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)

	// Layout
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colorMuted).
			PaddingBottom(1).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	SelectedItem = lipgloss.NewStyle().
			Foreground(colorSelected).
			Bold(true)

	NormalItem = lipgloss.NewStyle()

	// Help bar at the bottom
	HelpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	// Stats table
	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colorMuted)

	SuggestionStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Italic(true)

	// Step header in interactive mode
	StepStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			PaddingLeft(1).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeft(true).
			BorderForeground(colorPrimary)

	// Result summary
	SummaryStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess).
			MarginTop(1)
)
