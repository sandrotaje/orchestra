package tui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	colorBase     = lipgloss.Color("#1e1e2e")
	colorSurface0 = lipgloss.Color("#313244")
	colorSurface1 = lipgloss.Color("#45475a")
	colorText     = lipgloss.Color("#cdd6f4")
	colorSubtext  = lipgloss.Color("#a6adc8")
	colorOverlay  = lipgloss.Color("#6c7086")
	colorGreen    = lipgloss.Color("#a6e3a1")
	colorRed      = lipgloss.Color("#f38ba8")
	colorYellow   = lipgloss.Color("#f9e2af")
	colorBlue     = lipgloss.Color("#89b4fa")
	colorMauve    = lipgloss.Color("#cba6f7")
	colorPeach    = lipgloss.Color("#fab387")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMauve).
			MarginBottom(1)

	projectHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBlue)

	cardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSurface1).
			Padding(0, 1)

	selectedCardBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorMauve).
				Padding(0, 1)

	prNumberStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)

	checkSuccessStyle = lipgloss.NewStyle().Foreground(colorGreen)
	checkFailureStyle = lipgloss.NewStyle().Foreground(colorRed)
	checkPendingStyle = lipgloss.NewStyle().Foreground(colorYellow)
	checkSkippedStyle = lipgloss.NewStyle().Foreground(colorOverlay)

	additionsStyle = lipgloss.NewStyle().Foreground(colorGreen)
	deletionsStyle = lipgloss.NewStyle().Foreground(colorRed)

	approvedStyle       = lipgloss.NewStyle().Foreground(colorGreen)
	changesRequestStyle = lipgloss.NewStyle().Foreground(colorRed)
	mergedStyle         = lipgloss.NewStyle().Foreground(colorMauve).Bold(true)
	draftStyle          = lipgloss.NewStyle().Foreground(colorOverlay).Italic(true)
	prTitleStyle        = lipgloss.NewStyle().Foreground(colorText)
	detailStyle         = lipgloss.NewStyle().Foreground(colorSubtext)

	noPRStyle = lipgloss.NewStyle().Foreground(colorOverlay).Italic(true)

	authorBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBase).
				Background(colorMauve)

	reviewerBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBase).
				Background(colorBlue)

	confirmStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true).
			MarginTop(1)

	toastStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true).
			MarginTop(1)

	helpBarStyle = lipgloss.NewStyle().
			Foreground(colorOverlay).
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSurface0)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorOverlay)

	claudeWorkingStyle      = lipgloss.NewStyle().Foreground(colorYellow)
	claudeDoneStyle         = lipgloss.NewStyle().Foreground(colorGreen)
	claudeNotificationStyle = lipgloss.NewStyle().Foreground(colorPeach)

	modalOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorMauve).
				Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMauve).
			MarginBottom(1)

	filterMatchStyle = lipgloss.NewStyle().
				Foreground(colorText)

	filterDimStyle = lipgloss.NewStyle().
			Foreground(colorOverlay)
)
