package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gh "github.com/BobcatProgrammer/gh-notify-tidy/internal/github"
)

// Step represents which stage of the interactive flow we are in.
type Step int

const (
	StepLoading     Step = iota // fetching + checking staleness
	StepStats                   // show stats / overview
	StepBatchGroups             // triage stale groups
	StepConfirm                 // review pending actions
	StepApplying                // applying actions
	StepDone                    // finished
)

// Action the user chose for a group of notifications.
type Action int

const (
	ActionNone Action = iota
	ActionRead
	ActionDone
	ActionMute
	ActionSkip
)

func (a Action) String() string {
	switch a {
	case ActionRead:
		return "read"
	case ActionDone:
		return "done"
	case ActionMute:
		return "mute"
	case ActionSkip:
		return "skip"
	default:
		return ""
	}
}

// groupItem wraps a StaleGroup for display in the batch-groups list.
type groupItem struct {
	group  gh.StaleGroup
	action Action
}

func (i groupItem) Title() string {
	return fmt.Sprintf("%s  ·  %s  (%d)",
		i.group.Repo, string(i.group.StaleReason), len(i.group.Threads))
}

func (i groupItem) Description() string {
	titles := make([]string, 0, 3)
	for j, t := range i.group.Threads {
		if j >= 3 {
			titles = append(titles, fmt.Sprintf("… and %d more", len(i.group.Threads)-3))
			break
		}

		titles = append(titles, truncate(t.Subject.Title, 60))
	}

	action := ""
	if i.action != ActionNone {
		action = "  →  " + strings.ToUpper(i.action.String())
	}

	return strings.Join(titles, "  |  ") + action
}

func (i groupItem) FilterValue() string {
	return i.group.Repo + " " + string(i.group.StaleReason)
}

// pendingAction pairs a thread with the chosen action for the confirm step.
type pendingAction struct {
	thread gh.Thread
	action Action
}

// -- Messages

// loadedMsg is sent when the initial notification fetch completes.
type loadedMsg struct {
	threads     []gh.Thread
	stats       []gh.Stats
	viewerLogin string
	err         error
}

// staleCheckedMsg is sent when the staleness pass completes.
type staleCheckedMsg struct {
	groups []gh.StaleGroup
	err    error
}

// appliedMsg is sent when all actions have been applied.
type appliedMsg struct {
	done  int
	read  int
	muted int
	errs  []error
}

// Model is the root Bubble Tea model for the interactive flow.
type Model struct {
	client *gh.Client
	filter gh.Filter
	dryRun bool

	step    Step
	spinner spinner.Model

	loadingLabel string

	// Data
	allThreads  []gh.Thread
	stats       []gh.Stats
	viewerLogin string
	groups      []gh.StaleGroup

	// Per-step list widget
	list list.Model

	// Actions decided per group index (groupIdx → Action)
	groupActions map[int]Action

	// Confirm/apply state
	pending []pendingAction
	applied appliedMsg

	width  int
	height int
}

// New creates a new interactive Model.
func New(client *gh.Client, filter gh.Filter, dryRun bool) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		client:       client,
		filter:       filter,
		dryRun:       dryRun,
		step:         StepLoading,
		spinner:      s,
		loadingLabel: "Loading notifications…",
		groupActions: make(map[int]Action),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadNotifications())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case loadedMsg:
		if msg.err != nil {
			return m, tea.Quit
		}

		m.allThreads = msg.threads
		m.stats = msg.stats
		m.viewerLogin = msg.viewerLogin
		m.loadingLabel = fmt.Sprintf("Checking %d notification(s) for staleness…", len(msg.threads))
		return m, m.checkStale()

	case staleCheckedMsg:
		if msg.err != nil {
			return m, tea.Quit
		}

		m.groups = msg.groups
		m.step = StepStats
		return m, nil

	case appliedMsg:
		m.applied = msg
		m.step = StepDone
		return m, nil
	}

	// Delegate list updates when in the groups step.
	if m.step == StepBatchGroups {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.step {
	case StepLoading:
		return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), m.loadingLabel)

	case StepStats:
		return m.viewStats()

	case StepBatchGroups:
		return m.viewBatchGroups()

	case StepConfirm:
		return m.viewConfirm()

	case StepApplying:
		return m.viewApplying()

	case StepDone:
		return m.viewDone()
	}

	return ""
}

// -- Key handling

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	switch m.step {
	case StepStats:
		return m.handleStatsKey(msg)
	case StepBatchGroups:
		return m.handleGroupsKey(msg)
	case StepConfirm:
		return m.handleConfirmKey(msg)
	}

	return m, nil
}

func (m Model) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "n", " ":
		return m.enterBatchGroups(), nil
	}

	return m, nil
}

func (m Model) handleGroupsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		return m.setGroupAction(ActionRead), nil
	case "d":
		return m.setGroupAction(ActionDone), nil
	case "m":
		return m.setGroupAction(ActionMute), nil
	case "s", " ":
		return m.setGroupAction(ActionSkip), nil
	case "enter", "n":
		m.pending = m.buildPending()
		m.step = StepConfirm
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.step = StepApplying
		return m, m.applyActions()
	case "n", "esc":
		return m, tea.Quit
	}

	return m, nil
}

// -- Step transitions

func (m Model) enterBatchGroups() Model {
	if len(m.groups) == 0 {
		// No stale notifications — skip to done.
		m.step = StepDone

		return m
	}

	items := make([]list.Item, len(m.groups))
	for i, g := range m.groups {
		action := m.groupActions[i]
		if action == ActionNone {
			action = ActionDone // sensible default
		}

		m.groupActions[i] = action
		items[i] = groupItem{group: g, action: action}
	}

	delegate := list.NewDefaultDelegate()
	m.list = list.New(items, delegate, m.width, m.height-6)
	m.list.SetShowHelp(false)
	m.list.Title = fmt.Sprintf("Stale Notifications (%d groups)", len(m.groups))
	m.step = StepBatchGroups

	return m
}

func (m Model) setGroupAction(a Action) Model {
	idx := m.list.Index()
	if idx < 0 || idx >= len(m.groups) {
		return m
	}

	m.groupActions[idx] = a

	item := groupItem{group: m.groups[idx], action: a}
	m.list.SetItem(idx, item)

	return m
}

func (m Model) buildPending() []pendingAction {
	var out []pendingAction

	for idx, action := range m.groupActions {
		if action == ActionNone || action == ActionSkip {
			continue
		}

		if idx >= len(m.groups) {
			continue
		}

		for _, t := range m.groups[idx].Threads {
			out = append(out, pendingAction{thread: t, action: action})
		}
	}

	return out
}

// -- Views

func (m Model) viewStats() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Notification Statistics") + "\n\n")

	if len(m.stats) == 0 {
		b.WriteString(Muted.Render("No notifications found.") + "\n")
	} else {
		header := fmt.Sprintf("%-40s %6s %6s  %-20s  %s",
			"Repository", "Total", "Unread", "Top reason", "Suggestion")
		b.WriteString(TableHeader.Render(header) + "\n")

		for _, s := range m.stats {
			topReason := topKey(s.ByReason)
			sug := ""

			if s.Suggestion != "" {
				sug = SuggestionStyle.Render("→ " + s.Suggestion)
			}

			line := fmt.Sprintf("%-40s %6d %6d  %-20s  %s",
				truncate(s.Repo, 40), s.Total, s.Unread, topReason, sug)
			b.WriteString(line + "\n")
		}
	}

	staleCount := 0
	for _, g := range m.groups {
		staleCount += len(g.Threads)
	}

	b.WriteString("\n")

	if staleCount > 0 {
		b.WriteString(fmt.Sprintf(
			SuggestionStyle.Render("Found %d stale notification(s) in %d group(s) — press enter to triage"),
			staleCount, len(m.groups)) + "\n")
	} else {
		b.WriteString(Muted.Render("No stale notifications found.") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("press enter to continue · q to quit"))

	return b.String()
}

func (m Model) viewBatchGroups() string {
	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(
		"r=read  d=done  m=mute  s=skip  ↑↓=navigate  enter=review & apply  q=quit"))

	return b.String()
}

func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Review actions") + "\n\n")

	if len(m.pending) == 0 {
		b.WriteString(Muted.Render("Nothing to do — no actions selected.") + "\n\n")
		b.WriteString(HelpStyle.Render("press q to quit"))

		return b.String()
	}

	// Group pending by repo for readable display.
	repoOrder := []string{}
	byRepo := map[string][]pendingAction{}

	for _, p := range m.pending {
		repo := p.thread.Repository.FullName
		if _, ok := byRepo[repo]; !ok {
			repoOrder = append(repoOrder, repo)
		}

		byRepo[repo] = append(byRepo[repo], p)
	}

	for _, repo := range repoOrder {
		_, _ = fmt.Fprintf(&b, "  %s\n", Muted.Render(repo))

		for _, p := range byRepo[repo] {
			badge := actionBadge(p.action)
			_, _ = fmt.Fprintf(&b, "    %s  %s\n", badge, p.thread.Subject.Title)
		}
	}

	b.WriteString("\n")

	if m.dryRun {
		b.WriteString(BadgeWarning().Render("DRY RUN — no changes will be made") + "\n\n")
	}

	b.WriteString(HelpStyle.Render("y=apply  n=cancel"))

	return b.String()
}

func (m Model) viewApplying() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Applying…") + "\n\n")
	fmt.Fprintf(&b, "\n  %s working…\n", m.spinner.View())

	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Done!") + "\n\n")

	if len(m.groups) == 0 {
		b.WriteString(Muted.Render("No stale notifications found — inbox already clean.") + "\n")
	} else {
		b.WriteString(SummaryStyle.Render(fmt.Sprintf(
			"  Marked read: %d  |  Done: %d  |  Muted: %d",
			m.applied.read, m.applied.done, m.applied.muted,
		)) + "\n")
	}

	if len(m.applied.errs) > 0 {
		b.WriteString(BadgeDanger.Render(fmt.Sprintf("\n  %d error(s):", len(m.applied.errs))) + "\n")

		for _, e := range m.applied.errs {
			fmt.Fprintf(&b, "    %v\n", e)
		}
	}

	b.WriteString("\n" + HelpStyle.Render("press q to exit"))

	return b.String()
}

// -- Commands (Bubble Tea Cmds)

func (m Model) loadNotifications() tea.Cmd {
	return func() tea.Msg {
		// Fetch notifications and viewer login in sequence (both needed before stale check).
		threads, err := m.client.ListAll(m.filter)
		if err != nil {
			return loadedMsg{err: err}
		}

		viewerLogin, err := m.client.GetAuthenticatedUser()
		if err != nil {
			return loadedMsg{err: err}
		}

		stats := gh.ComputeStats(threads)

		return loadedMsg{threads: threads, stats: stats, viewerLogin: viewerLogin}
	}
}

func (m Model) checkStale() tea.Cmd {
	return func() tea.Msg {
		staleThreads := gh.StaleOnly(gh.CheckStalePR(m.client, m.allThreads, m.viewerLogin))
		groups := gh.GroupByRepoReason(staleThreads)

		return staleCheckedMsg{groups: groups}
	}
}

func (m Model) applyActions() tea.Cmd {
	return func() tea.Msg {
		var (
			done  int
			read  int
			muted int
			errs  []error
		)

		for _, p := range m.pending {
			if m.dryRun {
				switch p.action {
				case ActionRead:
					read++
				case ActionDone:
					done++
				case ActionMute:
					muted++
				}

				continue
			}

			var err error

			switch p.action {
			case ActionRead:
				err = m.client.MarkRead(p.thread.ID)
				if err == nil {
					read++
				}
			case ActionDone:
				err = m.client.Done(p.thread.ID)
				if err == nil {
					done++
				}
			case ActionMute:
				err = m.client.Mute(p.thread.ID)
				if err == nil {
					muted++
				}
			}

			if err != nil {
				errs = append(errs, err)
			}
		}

		return appliedMsg{done: done, read: read, muted: muted, errs: errs}
	}
}

// -- Helpers

func actionBadge(a Action) string {
	switch a {
	case ActionRead:
		return BadgeRead.Render("[READ]")
	case ActionDone:
		return BadgeDone.Render("[DONE]")
	case ActionMute:
		return BadgeMute.Render("[MUTE]")
	default:
		return BadgeSkip.Render("[SKIP]")
	}
}

// BadgeWarning returns a warning-coloured style (used outside the vars block
// to avoid initialisation order issues).
func BadgeWarning() lipgloss.Style {
	return BadgeMute
}

func topKey(m map[string]int) string {
	top, count := "", 0

	for k, v := range m {
		if v > count {
			top, count = k, v
		}
	}

	return top
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max-1] + "…"
}

// humanAge returns a human-readable age string for t, e.g. "3d ago".
func humanAge(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}
