package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"orchestra/internal/aggregator"
	"orchestra/internal/github"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time
type toastMsg struct{ text string }
type clearToastMsg struct{}
type clipboardMsg struct{ err error }
type errMsg struct{ err error }
type branchUpdatedMsg struct{}

type Model struct {
	agg            *aggregator.Aggregator
	groups         []aggregator.ProjectGroup
	cursor         int // flat index across all tabs
	creating       bool
	create         createModel
	confirmRemove  bool // true when showing remove confirmation
	lastRefreshAt  time.Time
	width          int
	height         int
	toast          string // status message shown at the bottom
	filtering      bool   // true when in search/filter mode
	filterText     string // current filter query
	loading        bool   // true until first data arrives
	spinner        spinner.Model
}

func NewModel(agg *aggregator.Aggregator) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorMauve)
	return Model{
		agg:     agg,
		width:   80,
		height:  24,
		loading: true,
		spinner: s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchDataCmd(m.agg), tickCmd(), m.spinner.Tick)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type dataMsg struct {
	groups []aggregator.ProjectGroup
}

func fetchDataCmd(agg *aggregator.Aggregator) tea.Cmd {
	return func() tea.Msg {
		groups := agg.Refresh()
		return dataMsg{groups: groups}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			fetchDataCmd(m.agg),
			tickCmd(),
		)

	case dataMsg:
		m.groups = msg.groups
		m.loading = false
		// Clamp cursor
		total := m.totalTabs()
		if m.cursor >= total {
			m.cursor = max(0, total-1)
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tabCreatedMsg:
		m.creating = false
		return m, nil

	case tabRemovedMsg:
		m.confirmRemove = false
		return m, fetchDataCmd(m.agg)

	case errMsg:
		m.toast = msg.err.Error()
		return m, clearToastCmd()

	case clipboardMsg:
		if msg.err != nil {
			m.toast = "Failed to copy to clipboard"
		} else {
			m.toast = "Copied PR URL to clipboard"
		}
		return m, clearToastCmd()

	case branchUpdatedMsg:
		m.toast = "Branch updated"
		return m, tea.Batch(clearToastCmd(), fetchDataCmd(m.agg))

	case toastMsg:
		m.toast = msg.text
		return m, clearToastCmd()

	case clearToastMsg:
		m.toast = ""
		return m, nil

	case tea.KeyMsg:
		if m.creating {
			return m.updateCreate(msg)
		}
		if m.confirmRemove {
			return m.updateConfirmRemove(msg)
		}
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateDashboard(msg)
	}

	return m, nil
}

func (m Model) updateCreate(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.create, cmd = m.create.Update(msg)
	if m.create.step == -1 {
		m.creating = false
	}
	return m, cmd
}

func (m Model) updateDashboard(msg tea.KeyMsg) (Model, tea.Cmd) {
	action := parseKey(msg)
	total := m.totalTabs()

	switch action {
	case keyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case keyDown:
		if m.cursor < total-1 {
			m.cursor++
		}
	case keyTop:
		m.cursor = 0
	case keyBottom:
		if total > 0 {
			m.cursor = total - 1
		}
	case keyNextGroup:
		m.cursor = m.nextGroupStart(m.cursor)
	case keyPrevGroup:
		m.cursor = m.prevGroupStart(m.cursor)
	case keyEnter:
		if tab := m.selectedTab(); tab != nil {
			return m, activateTabCmd(tab.TabID)
		}
	case keyNew:
		projects := m.agg.ListProjects()
		m.creating = true
		m.create = newCreateModel(projects, m.width, m.height)
		return m, nil
	case keyOpenPR:
		if tab := m.selectedTab(); tab != nil && tab.PR != nil {
			return m, openURLCmd(tab.PR.URL)
		}
	case keyRemove:
		if tab := m.selectedTab(); tab != nil {
			m.confirmRemove = true
			return m, nil
		}
	case keyFilter:
		m.filtering = true
		m.filterText = ""
		return m, nil
	case keyCopy:
		if tab := m.selectedTab(); tab != nil && tab.PR != nil {
			return m, copyToClipboardCmd(tab.PR.URL)
		}
	case keyUpdateBranch:
		if tab := m.selectedTab(); tab != nil && tab.PR != nil && tab.PR.State != "MERGED" && tab.PR.State != "CLOSED" {
			m.toast = "Updating branch..."
			return m, updateBranchCmd(tab.RepoPath, tab.PR.Number)
		}
	case keyEscape:
		if m.filterText != "" {
			m.filterText = ""
			m.cursor = 0
			return m, nil
		}
	case keyRefresh:
		m.agg.ClearPRCache()
		m.toast = "Refreshing..."
		return m, tea.Batch(fetchDataCmd(m.agg), clearToastCmd())
	case keyQuit:
		return m, tea.Quit
	}

	return m, nil
}

type tabRemovedMsg struct{}

func (m Model) updateFilter(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterText = ""
		return m, nil
	case "enter":
		m.filtering = false
		// Keep filterText active so results stay filtered
		return m, nil
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
		} else {
			m.filtering = false
		}
		m.cursor = 0
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.filterText += msg.String()
			m.cursor = 0
		}
		return m, nil
	}
}

func (m Model) updateConfirmRemove(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if tab := m.selectedTab(); tab != nil && len(tab.PaneIDs) > 0 {
			m.confirmRemove = false
			return m, removeTabCmd(tab.PaneIDs, tab.WorkDir)
		}
		m.confirmRemove = false
	case "n", "esc":
		m.confirmRemove = false
	}
	return m, nil
}

func removeTabCmd(paneIDs []int, workDir string) tea.Cmd {
	return func() tea.Msg {
		// Close the tab by killing all its panes
		for _, pid := range paneIDs {
			exec.Command("wezterm", "cli", "kill-pane", "--pane-id", fmt.Sprint(pid)).Run()
		}

		// Run ww rm in the worktree directory
		if workDir != "" {
			cmd := exec.Command("ww", "rm")
			cmd.Dir = workDir
			if err := cmd.Run(); err != nil {
				return errMsg{err: fmt.Errorf("failed to remove worktree: %w", err)}
			}
		}

		return tabRemovedMsg{}
	}
}

func activateTabCmd(tabID int) tea.Cmd {
	return func() tea.Msg {
		if err := exec.Command("wezterm", "cli", "activate-tab", "--tab-id", fmt.Sprint(tabID)).Run(); err != nil {
			return errMsg{err: fmt.Errorf("failed to switch tab: %w", err)}
		}
		return nil
	}
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := exec.Command("open", url).Run(); err != nil {
			return errMsg{err: fmt.Errorf("failed to open URL: %w", err)}
		}
		return nil
	}
}

func copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		err := cmd.Run()
		return clipboardMsg{err: err}
	}
}

func updateBranchCmd(repoDir string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		if err := github.UpdatePRBranch(repoDir, prNumber); err != nil {
			return errMsg{err: fmt.Errorf("update branch: %w", err)}
		}
		return branchUpdatedMsg{}
	}
}

func clearToastCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearToastMsg{}
	})
}

func (m Model) View() string {
	if m.creating {
		return m.renderCreateOverlay()
	}

	var header strings.Builder
	header.WriteString(titleStyle.Render("♫ Orchestra"))
	header.WriteString("\n")

	// Loading spinner
	if m.loading {
		header.WriteString("\n")
		header.WriteString(m.spinner.View() + " Loading tabs...")
		header.WriteString("\n")
		return header.String()
	}

	// Summary stats
	if len(m.groups) > 0 {
		header.WriteString(m.renderSummaryStats())
		header.WriteString("\n")
		sepWidth := m.width - 2
		if sepWidth > 80 {
			sepWidth = 80
		}
		separator := lipgloss.NewStyle().Foreground(colorSurface1).Render(strings.Repeat("─", sepWidth))
		header.WriteString(separator)
		header.WriteString("\n")
	}

	// Filter bar
	if m.filtering || m.filterText != "" {
		if m.filtering {
			header.WriteString(filterMatchStyle.Render("/ " + m.filterText + "█"))
		} else {
			header.WriteString(filterMatchStyle.Render("/ " + m.filterText))
		}
		header.WriteString("\n")
	}

	// Build scrollable body
	var body strings.Builder
	groups := m.filteredGroups()

	if len(m.groups) == 0 {
		body.WriteString(noPRStyle.Render("No WezTerm tabs found. Is WezTerm running?"))
		body.WriteString("\n\n")
		body.WriteString(helpBarStyle.Render("  Press n to create your first tab, or r to refresh"))
		body.WriteString("\n")
	} else if len(groups) == 0 && m.filterText != "" {
		body.WriteString(noPRStyle.Render("No matching tabs"))
		body.WriteString("\n")
	}

	// Track line range of the selected card for auto-scroll
	selectedStart, selectedEnd := -1, -1
	flatIdx := 0
	currentLine := 0

	for _, group := range groups {
		accent := projectHeaderStyle.Render("▍")
		groupLabel := fmt.Sprintf(" %s (%d) ", group.Name, len(group.Tabs))
		dashes := strings.Repeat("─", max(0, m.width-lipgloss.Width(groupLabel)-3))
		groupHeader := accent + projectHeaderStyle.Render(groupLabel) + projectHeaderStyle.Render(dashes)
		body.WriteString(groupHeader)
		body.WriteString("\n\n")
		currentLine += 2

		for _, tab := range group.Tabs {
			card := renderCard(tab, flatIdx == m.cursor, m.width)
			cardLines := strings.Count(card, "\n") + 1
			if flatIdx == m.cursor {
				selectedStart = currentLine
				selectedEnd = currentLine + cardLines
			}
			body.WriteString(card)
			body.WriteString("\n")
			currentLine += cardLines + 1
			flatIdx++
		}
		body.WriteString("\n")
		currentLine++
	}

	// Build footer
	var footer strings.Builder
	if m.confirmRemove {
		if tab := m.selectedTab(); tab != nil {
			confirm := confirmStyle.Render(fmt.Sprintf("Remove %s/%s? (y/n)", tab.Project, tab.Branch))
			footer.WriteString(confirm)
			footer.WriteString("\n")
		}
	}
	if m.toast != "" {
		footer.WriteString(toastStyle.Render(m.toast))
		footer.WriteString("\n")
	}
	refreshIn := m.refreshCountdown()
	footer.WriteString(m.renderHelpBar(refreshIn))

	// Apply viewport scrolling
	headerStr := header.String()
	footerStr := footer.String()
	headerLines := strings.Count(headerStr, "\n")
	footerLines := strings.Count(footerStr, "\n") + 1
	viewportHeight := m.height - headerLines - footerLines
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	// Auto-scroll to keep selected card visible
	scrollOff := 0
	if selectedStart >= 0 {
		if selectedEnd > scrollOff+viewportHeight {
			scrollOff = selectedEnd - viewportHeight
		}
		if selectedStart < scrollOff {
			scrollOff = selectedStart
		}
	}
	if scrollOff < 0 {
		scrollOff = 0
	}

	// Slice body lines to viewport
	bodyLines := strings.Split(body.String(), "\n")
	totalBodyLines := len(bodyLines)
	if scrollOff > totalBodyLines {
		scrollOff = totalBodyLines
	}
	end := scrollOff + viewportHeight
	if end > totalBodyLines {
		end = totalBodyLines
	}
	visibleBody := strings.Join(bodyLines[scrollOff:end], "\n")

	// Scroll indicator
	scrollIndicator := ""
	if scrollOff > 0 {
		scrollIndicator = detailStyle.Render("▲ more") + "\n"
	}
	bottomIndicator := ""
	if end < totalBodyLines-1 {
		bottomIndicator = "\n" + detailStyle.Render("▼ more")
	}

	return headerStr + scrollIndicator + visibleBody + bottomIndicator + "\n" + footerStr
}

func (m Model) renderHelpBar(refreshIn string) string {
	type entry struct {
		key  string
		desc string
	}
	keys := []entry{
		{"j/k", "navigate"},
		{"l/Enter", "switch"},
		{"n", "new"},
		{"x", "remove"},
		{"o", "open PR"},
		{"y", "copy"},
		{"u", "update"},
		{"g/G", "top/end"},
		{"Tab", "groups"},
		{"/", "filter"},
		{"r", "refresh"},
		{"q", "quit"},
	}

	if m.width >= 100 {
		// Full help bar with pill-styled keys
		parts := make([]string, len(keys))
		for i, k := range keys {
			key := helpKeyStyle.Render(" " + k.key + " ")
			desc := helpDescStyle.Render(k.desc)
			parts[i] = key + " " + desc
		}
		return helpBarStyle.Render(strings.Join(parts, "  ") + "  " + refreshIn)
	}

	// Compact help bar — just key pills
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = helpKeyStyle.Render(" " + k.key + " ")
	}
	return helpBarStyle.Render(strings.Join(parts, " ") + "  " + refreshIn)
}

func (m Model) renderCreateOverlay() string {
	// Center the modal
	modal := m.create.View()
	modalHeight := lipgloss.Height(modal)
	modalWidth := lipgloss.Width(modal)

	padTop := max(0, (m.height-modalHeight)/2)
	padLeft := max(0, (m.width-modalWidth)/2)

	return strings.Repeat("\n", padTop) +
		strings.Repeat(" ", padLeft) +
		strings.ReplaceAll(modal, "\n", "\n"+strings.Repeat(" ", padLeft))
}

func (m Model) totalTabs() int {
	n := 0
	for _, g := range m.filteredGroups() {
		n += len(g.Tabs)
	}
	return n
}

func (m Model) refreshCountdown() string {
	oldest := m.agg.OldestPRFetch()
	if oldest.IsZero() {
		return "refreshing..."
	}
	ttl := aggregator.PRCacheTTL()
	elapsed := time.Since(oldest)
	remaining := ttl - elapsed
	if remaining <= 0 {
		return "refreshing..."
	}
	mins := int(remaining.Minutes())
	secs := int(remaining.Seconds()) % 60
	return fmt.Sprintf("⟳ %d:%02d", mins, secs)
}

func (m Model) filteredGroups() []aggregator.ProjectGroup {
	if m.filterText == "" {
		return m.groups
	}
	query := strings.ToLower(m.filterText)
	var result []aggregator.ProjectGroup
	for _, g := range m.groups {
		var matched []aggregator.DashboardTab
		for _, tab := range g.Tabs {
			if strings.Contains(strings.ToLower(tab.Branch), query) ||
				strings.Contains(strings.ToLower(tab.Project), query) ||
				(tab.PR != nil && strings.Contains(strings.ToLower(tab.PR.Title), query)) {
				matched = append(matched, tab)
			}
		}
		if len(matched) > 0 {
			result = append(result, aggregator.ProjectGroup{Name: g.Name, Tabs: matched})
		}
	}
	return result
}

func (m Model) renderSummaryStats() string {
	totalTabs, totalPRs, failing, ready := 0, 0, 0, 0
	for _, g := range m.groups {
		for _, tab := range g.Tabs {
			totalTabs++
			if tab.PR != nil {
				totalPRs++
				status, _ := prStatus(tab.PR)
				switch {
				case strings.Contains(status, "failing") || strings.Contains(status, "conflict"):
					failing++
				case status == "Ready to merge":
					ready++
				}
			}
		}
	}
	parts := []string{
		detailStyle.Render(fmt.Sprintf("%d tabs", totalTabs)),
		detailStyle.Render(fmt.Sprintf("%d PRs", totalPRs)),
	}
	if failing > 0 {
		parts = append(parts, checkFailureStyle.Render(fmt.Sprintf("%d failing", failing)))
	}
	if ready > 0 {
		parts = append(parts, checkSuccessStyle.Render(fmt.Sprintf("%d ready", ready)))
	}
	return strings.Join(parts, detailStyle.Render(" · "))
}

func (m Model) nextGroupStart(cursor int) int {
	idx := 0
	for _, g := range m.groups {
		groupStart := idx
		idx += len(g.Tabs)
		// Find the first group whose start is past the cursor
		if groupStart > cursor {
			return groupStart
		}
	}
	return cursor // already at last group
}

func (m Model) prevGroupStart(cursor int) int {
	idx := 0
	prevStart := 0
	for _, g := range m.groups {
		if idx >= cursor {
			return prevStart
		}
		prevStart = idx
		idx += len(g.Tabs)
	}
	return prevStart
}

func (m Model) selectedTab() *aggregator.DashboardTab {
	idx := 0
	groups := m.filteredGroups()
	for _, g := range groups {
		for i := range g.Tabs {
			if idx == m.cursor {
				return &g.Tabs[i]
			}
			idx++
		}
	}
	return nil
}


