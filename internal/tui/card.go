package tui

import (
	"fmt"
	"strings"
	"time"

	"orchestra/internal/aggregator"
	"orchestra/internal/github"

	"github.com/charmbracelet/lipgloss"
)

func renderCard(tab aggregator.DashboardTab, selected bool, width int) string {
	cardWidth := width - 6 // account for margins + border padding
	if cardWidth < 40 {
		cardWidth = 40
	}

	// Card title: claude icon + branch
	titleIcon := claudeIcon(tab.ClaudeStatus)
	title := titleIcon + " " + tab.Branch
	if tab.IsActive {
		title += " *"
	}

	// Row 1: PR info
	var prLine string
	if tab.PR != nil {
		prLine = renderPRLine(tab.PR, cardWidth)
	} else {
		prLine = noPRStyle.Render("No PR")
	}

	// Assemble content
	var content strings.Builder
	content.WriteString(prLine)

	// Render with appropriate border
	style := cardBorder
	if selected {
		style = selectedCardBorder
	}
	style = style.Width(cardWidth)

	card := style.Render(content.String())

	// Inject title into the top border
	card = injectBorderTitle(card, title, selected)

	// Indent each line of the card
	lines := strings.Split(card, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}

	return strings.Join(lines, "\n")
}

func injectBorderTitle(card, title string, selected bool) string {
	lines := strings.Split(card, "\n")
	if len(lines) == 0 {
		return card
	}

	// Replace the top border with: ╭─ title ──────╮
	topLine := lines[0]

	// Measure the visual width of the top line (strips ANSI codes)
	topWidth := lipgloss.Width(topLine)
	if topWidth < 4 {
		return card
	}

	borderColor := colorSurface1
	if selected {
		borderColor = colorMauve
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	titleStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)
	if selected {
		titleStyle = titleStyle.Foreground(colorMauve)
	}

	styledTitle := titleStyle.Render(title)
	titleLen := lipgloss.Width(title)

	// Use matching border characters: rounded (╭─╮) vs thick (┏━┓)
	cornerL, dash, cornerR := "╭", "─", "╮"

	// Rebuild: ╭─ title ─...─╮
	// Width:   2  1  titleLen  1  dashCount  1 = titleLen + dashCount + 5
	dashCount := topWidth - titleLen - 5
	if dashCount < 1 {
		dashCount = 1
	}

	newTop := borderStyle.Render(cornerL+dash) + " " + styledTitle + " " + borderStyle.Render(strings.Repeat(dash, dashCount)+cornerR)
	lines[0] = newTop

	return strings.Join(lines, "\n")
}

func claudeIcon(status string) string {
	switch status {
	case "working":
		return claudeWorkingStyle.Render("⏳")
	case "done":
		return claudeDoneStyle.Render("✅")
	case "notification":
		return claudeNotificationStyle.Render("💬")
	default:
		return "  "
	}
}

// prStatus derives the overall actionable status from PR data.
// Uses GitHub's reviewDecision as the authoritative review status.
func prStatus(pr *github.PRInfo) (label string, style lipgloss.Style) {
	fail, pending := 0, 0
	for _, c := range pr.Checks {
		switch c.Status {
		case github.CheckFailure:
			fail++
		case github.CheckPending:
			pending++
		}
	}

	switch {
	case pr.State == "MERGED":
		return "Merged", mergedStyle
	case pr.State == "CLOSED":
		return "Closed", checkFailureStyle
	case pr.Draft:
		return "Draft", draftStyle
	case fail > 0:
		return fmt.Sprintf("Checks failing (%d)", fail), checkFailureStyle
	case pr.Mergeable == "CONFLICTING":
		return "Merge conflicts", checkFailureStyle
	case pr.MergeStateStatus == "BEHIND":
		return "Needs update", checkPendingStyle
	case pr.ReviewDecision == "CHANGES_REQUESTED":
		return "Changes requested", changesRequestStyle
	case pr.ReviewDecision == "REVIEW_REQUIRED":
		return "Review required", checkPendingStyle
	case pending > 0:
		return fmt.Sprintf("Checks running (%d/%d)", len(pr.Checks)-pending, len(pr.Checks)), checkPendingStyle
	default:
		return "Ready to merge", checkSuccessStyle
	}
}

func formatAge(timestamp string) string {
	if timestamp == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ""
	}
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
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	}
}

func renderPRLine(pr *github.PRInfo, maxWidth int) string {
	// Line 1: PR #N → base  STATUS
	status, statusStyle := prStatus(pr)
	baseBranch := pr.Base
	if len(baseBranch) > 15 {
		baseBranch = baseBranch[:14] + "…"
	}
	baseStyle := detailStyle
	baseLabel := "→ " + baseBranch
	if pr.Mergeable == "CONFLICTING" {
		baseStyle = checkFailureStyle
		baseLabel = "→ " + baseBranch + " ✘"
	}
	line1 := prNumberStyle.Render(fmt.Sprintf("PR #%d", pr.Number)) + " " + baseStyle.Render(baseLabel) + "  " + statusStyle.Render(status)

	// Line 1.5: PR title (truncated to fit)
	var titleLine string
	if pr.Title != "" {
		title := pr.Title
		// Leave room for padding inside the card
		titleMax := maxWidth - 4
		if titleMax > 0 && len(title) > titleMax {
			title = title[:titleMax-1] + "…"
		}
		titleLine = prTitleStyle.Render(title)
	}

	// Line 2: details — checks, reviews, diff
	var details []string

	if bar := renderChecksBar(pr.Checks, 12); bar != "" {
		details = append(details, bar)
	}

	if rev := renderReviewers(pr.Reviews); rev != "" {
		details = append(details, rev)
	}

	diff := fmt.Sprintf("%s %s %s",
		additionsStyle.Render(fmt.Sprintf("+%d", pr.DiffStat.Additions)),
		deletionsStyle.Render(fmt.Sprintf("-%d", pr.DiffStat.Deletions)),
		detailStyle.Render(fmt.Sprintf("%df", pr.DiffStat.Files)),
	)
	details = append(details, diff)

	if age := formatAge(pr.UpdatedAt); age != "" {
		details = append(details, detailStyle.Render(age))
	}

	line2 := strings.Join(details, detailStyle.Render("  "))

	result := line1
	if titleLine != "" {
		result += "\n" + titleLine
	}
	result += "\n" + line2
	return result
}

func renderChecksBar(checks []github.Check, maxBarWidth int) string {
	if len(checks) == 0 {
		return ""
	}
	pass, fail, pending := 0, 0, 0
	for _, c := range checks {
		switch c.Status {
		case github.CheckSuccess:
			pass++
		case github.CheckFailure:
			fail++
		case github.CheckPending:
			pending++
		}
	}
	total := len(checks)

	// Calculate proportional segment widths
	passW := pass * maxBarWidth / total
	failW := fail * maxBarWidth / total
	pendW := pending * maxBarWidth / total
	emptyW := maxBarWidth - passW - failW - pendW

	var bar strings.Builder
	if passW > 0 {
		bar.WriteString(checkSuccessStyle.Render(strings.Repeat("█", passW)))
	}
	if failW > 0 {
		bar.WriteString(checkFailureStyle.Render(strings.Repeat("█", failW)))
	}
	if pendW > 0 {
		bar.WriteString(checkPendingStyle.Render(strings.Repeat("█", pendW)))
	}
	if emptyW > 0 {
		bar.WriteString(lipgloss.NewStyle().Foreground(colorSurface1).Render(strings.Repeat("░", emptyW)))
	}

	bar.WriteString(detailStyle.Render(fmt.Sprintf(" %d/%d", pass, total)))
	return bar.String()
}

func renderReviewers(reviews []github.Review) string {
	// Deduplicate by author, keeping latest state; only APPROVED and CHANGES_REQUESTED
	seen := make(map[string]github.ReviewState)
	var order []string
	for _, r := range reviews {
		if r.State != github.ReviewApproved && r.State != github.ReviewChangesRequested {
			continue
		}
		if _, exists := seen[r.Author]; !exists {
			order = append(order, r.Author)
		}
		seen[r.Author] = r.State
	}
	if len(order) == 0 {
		return ""
	}

	limit := 3
	var parts []string
	for i, author := range order {
		if i >= limit {
			break
		}
		name := author
		if len(name) > 8 {
			name = name[:7] + "…"
		}
		switch seen[author] {
		case github.ReviewApproved:
			parts = append(parts, approvedStyle.Render("✓ "+name))
		case github.ReviewChangesRequested:
			parts = append(parts, changesRequestStyle.Render("✗ "+name))
		}
	}
	if len(order) > limit {
		parts = append(parts, detailStyle.Render(fmt.Sprintf("+%d", len(order)-limit)))
	}
	return strings.Join(parts, " ")
}
