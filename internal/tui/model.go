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
	StepLoading   Step = iota // fetching notifications
	StepStats                 // show stats / worst offenders
	StepOld                   // old notifications (>N days)
	StepClosedPRs             // notifications for closed/merged PRs
	StepReadDone              // already-read notifications
	StepConfirm               // review pending actions
	StepApplying              // applying actions
	StepDone                  // finished
)

// Action the user chose for a notification.
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

// notifItem wraps a Thread for display in a bubbles/list.
type notifItem struct {
	thread gh.Thread
	action Action
}

func (i notifItem) Title() string {
	return fmt.Sprintf("[%s] %s", i.thread.Repository.FullName, i.thread.Subject.Title)
}

func (i notifItem) Description() string {
	reason := i.thread.Reason
	typ := i.thread.Subject.Type
	age := humanAge(i.thread.UpdatedAt)

	action := ""
	if i.action != ActionNone {
		action = " → " + strings.ToUpper(i.action.String())
	}

	return fmt.Sprintf("%s · %s · %s%s", typ, reason, age, action)
}

func (i notifItem) FilterValue() string {
	return i.thread.Repository.FullName + " " + i.thread.Subject.Title
}

// pendingAction pairs a thread with the chosen action for the confirm step.
type pendingAction struct {
	thread gh.Thread
	action Action
}

// loadedMsg is sent when the initial notification fetch completes.
type loadedMsg struct {
	threads []gh.Thread
	stats   []gh.Stats
	err     error
}

// appliedMsg is sent when all actions have been applied.
type appliedMsg struct {
	done  int
	read  int
	muted int
	errs  []error
}

// progressMsg is sent after each individual action during apply.
type progressMsg struct {
	threadID string
	action   Action
	err      error
}

// Model is the root Bubble Tea model for the interactive flow.
type Model struct {
	client  *gh.Client
	filter  gh.Filter
	dryRun  bool
	oldDays int

	step    Step
	spinner spinner.Model

	// Data
	allThreads  []gh.Thread
	stats       []gh.Stats
	oldThreads  []gh.Thread
	closedPRs   []gh.Thread
	readThreads []gh.Thread

	// Per-step list widgets
	list list.Model

	// Actions decided per thread (threadID → Action)
	decisions map[string]Action

	// Confirm/apply state
	pending  []pendingAction
	progress []string
	applied  appliedMsg

	width  int
	height int
}

// New creates a new interactive Model.
func New(client *gh.Client, filter gh.Filter, dryRun bool, oldDays int) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		client:    client,
		filter:    filter,
		dryRun:    dryRun,
		oldDays:   oldDays,
		step:      StepLoading,
		spinner:   s,
		decisions: make(map[string]Action),
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
			// Surface error and quit gracefully.
			return m, tea.Quit
		}

		m.allThreads = msg.threads
		m.stats = msg.stats
		m.oldThreads = filterOld(msg.threads, m.oldDays)
		m.closedPRs = filterClosedPRs(msg.threads)
		m.readThreads = filterRead(msg.threads)
		m.step = StepStats
		return m, nil

	case progressMsg:
		status := fmt.Sprintf("  %s → %s", msg.threadID, msg.action)
		if msg.err != nil {
			status += fmt.Sprintf(" ERROR: %v", msg.err)
		}

		m.progress = append(m.progress, status)
		return m, nil

	case appliedMsg:
		m.applied = msg
		m.step = StepDone
		return m, nil

	case list.Model:
		m.list = msg
	}

	// Delegate list updates when in a list step.
	if m.inListStep() {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.step {
	case StepLoading:
		return fmt.Sprintf("\n  %s Loading notifications…\n", m.spinner.View())

	case StepStats:
		return m.viewStats()

	case StepOld, StepClosedPRs, StepReadDone:
		return m.viewList()

	case StepConfirm:
		return m.viewConfirm()

	case StepApplying:
		return m.viewApplying()

	case StepDone:
		return m.viewDone()
	}

	return ""
}

// ── Key handling ────────────────────────────────────────────────────────────

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	switch m.step {
	case StepStats:
		return m.handleStatsKey(msg)
	case StepOld, StepClosedPRs, StepReadDone:
		return m.handleListKey(msg)
	case StepConfirm:
		return m.handleConfirmKey(msg)
	}

	return m, nil
}

func (m Model) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "n", " ":
		return m.enterStep(StepOld)
	}

	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		return m.setCurrentAction(ActionRead)
	case "d":
		return m.setCurrentAction(ActionDone)
	case "m":
		return m.setCurrentAction(ActionMute)
	case "s", " ":
		return m.setCurrentAction(ActionSkip)
	case "enter", "n":
		// Advance to next step or confirm.
		return m.advanceStep()
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

// ── Step transitions ─────────────────────────────────────────────────────────

func (m Model) enterStep(s Step) (Model, tea.Cmd) {
	m.step = s
	var threads []gh.Thread

	switch s {
	case StepOld:
		threads = m.oldThreads
	case StepClosedPRs:
		threads = m.closedPRs
	case StepReadDone:
		threads = m.readThreads
	}

	if len(threads) == 0 {
		return m.advanceStep()
	}

	items := make([]list.Item, len(threads))
	for i, t := range threads {
		items[i] = notifItem{thread: t, action: m.decisions[t.ID]}
	}

	delegate := list.NewDefaultDelegate()
	m.list = list.New(items, delegate, m.width, m.height-6)
	m.list.SetShowHelp(false)
	m.list.Title = stepTitle(s)

	return m, nil
}

func (m Model) advanceStep() (Model, tea.Cmd) {
	switch m.step {
	case StepStats:
		return m.enterStep(StepOld)
	case StepOld:
		return m.enterStep(StepClosedPRs)
	case StepClosedPRs:
		return m.enterStep(StepReadDone)
	default:
		// Build pending list and go to confirm.
		m.pending = m.buildPending()
		m.step = StepConfirm
		return m, nil
	}
}

func (m Model) setCurrentAction(a Action) (Model, tea.Cmd) {
	if item, ok := m.list.SelectedItem().(notifItem); ok {
		m.decisions[item.thread.ID] = a
		item.action = a
		// Re-render the item in the list.
		idx := m.list.Index()
		m.list.SetItem(idx, item)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(nil)

	return m, cmd
}

func (m Model) buildPending() []pendingAction {
	var out []pendingAction

	for id, action := range m.decisions {
		if action == ActionNone || action == ActionSkip {
			continue
		}

		for _, t := range m.allThreads {
			if t.ID == id {
				out = append(out, pendingAction{thread: t, action: action})
				break
			}
		}
	}

	return out
}

func (m Model) inListStep() bool {
	return m.step == StepOld || m.step == StepClosedPRs || m.step == StepReadDone
}

// ── Views ────────────────────────────────────────────────────────────────────

func (m Model) viewStats() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Notification Statistics") + "\n\n")

	if len(m.stats) == 0 {
		b.WriteString(Muted.Render("No notifications found.") + "\n")
	} else {
		header := fmt.Sprintf("%-40s %6s %6s  %-30s  %s",
			"Repository", "Total", "Unread", "Top reason", "Suggestion")
		b.WriteString(TableHeader.Render(header) + "\n")

		for _, s := range m.stats {
			topReason := topKey(s.ByReason)
			sug := ""

			if s.Suggestion != "" {
				sug = SuggestionStyle.Render("→ " + s.Suggestion)
			}

			line := fmt.Sprintf("%-40s %6d %6d  %-30s  %s",
				truncate(s.Repo, 40), s.Total, s.Unread, topReason, sug)
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("press enter to start cleaning up · q to quit"))

	return b.String()
}

func (m Model) viewList() string {
	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("r=read  d=done  m=mute  s=skip  enter=next step  q=quit"))

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

	for _, p := range m.pending {
		badge := actionBadge(p.action)
		fmt.Fprintf(&b, "  %s  %s / %s\n",
			badge,
			Muted.Render(p.thread.Repository.FullName),
			p.thread.Subject.Title,
		)
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

	for _, line := range m.progress {
		b.WriteString(line + "\n")
	}

	fmt.Fprintf(&b, "\n  %s working…\n", m.spinner.View())

	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Done!") + "\n\n")
	b.WriteString(SummaryStyle.Render(fmt.Sprintf(
		"  Marked read: %d  |  Done: %d  |  Muted: %d",
		m.applied.read, m.applied.done, m.applied.muted,
	)) + "\n")

	if len(m.applied.errs) > 0 {
		b.WriteString(BadgeDanger.Render(fmt.Sprintf("\n  %d error(s):", len(m.applied.errs))) + "\n")

		for _, e := range m.applied.errs {
			fmt.Fprintf(&b, "    %v\n", e)
		}
	}

	b.WriteString("\n" + HelpStyle.Render("press q to exit"))

	return b.String()
}

// ── Commands ─────────────────────────────────────────────────────────────────

func (m Model) loadNotifications() tea.Cmd {
	return func() tea.Msg {
		threads, err := m.client.ListAll(m.filter)
		if err != nil {
			return loadedMsg{err: err}
		}

		stats := gh.ComputeStats(threads)

		return loadedMsg{threads: threads, stats: stats}
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

// ── Helpers ──────────────────────────────────────────────────────────────────

func stepTitle(s Step) string {
	switch s {
	case StepOld:
		return "Old Notifications"
	case StepClosedPRs:
		return "Closed / Merged PR Notifications"
	case StepReadDone:
		return "Already-read Notifications"
	default:
		return ""
	}
}

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

func filterOld(threads []gh.Thread, days int) []gh.Thread {
	cutoff := cutoffDuration(days)
	var out []gh.Thread

	for _, t := range threads {
		if t.UpdatedAt.Before(cutoff) {
			out = append(out, t)
		}
	}

	return out
}

func filterClosedPRs(threads []gh.Thread) []gh.Thread {
	var out []gh.Thread

	for _, t := range threads {
		if t.Subject.Type == "PullRequest" && !t.Unread {
			out = append(out, t)
		}
	}

	return out
}

func filterRead(threads []gh.Thread) []gh.Thread {
	var out []gh.Thread

	for _, t := range threads {
		if !t.Unread {
			out = append(out, t)
		}
	}

	return out
}

func cutoffDuration(days int) time.Time {
	return time.Now().AddDate(0, 0, -days)
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
