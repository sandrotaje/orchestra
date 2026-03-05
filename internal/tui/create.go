package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type createStep int

const (
	stepPickProject createStep = iota
	stepEnterBranch
)

type createModel struct {
	step         createStep
	projects     []string // full paths
	filtered     []string
	filter       string
	cursor       int
	selectedPath string
	branchInput  textinput.Model
	width        int
	height       int
}

func newCreateModel(projects []string, width, height int) createModel {
	ti := textinput.New()
	ti.Placeholder = "branch name or PR number"
	ti.CharLimit = 120
	ti.Width = 50

	return createModel{
		step:     stepPickProject,
		projects: projects,
		filtered: projects,
		width:    width,
		height:   height,
		branchInput: ti,
	}
}

type tabCreatedMsg struct{}

func (m createModel) Update(msg tea.Msg) (createModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			if m.step == stepEnterBranch {
				// Go back to project picker instead of cancelling
				m.step = stepPickProject
				m.branchInput.Blur()
				return m, nil
			}
			// Cancel from project picker
			m.step = -1
			return m, nil
		}

		if m.step == stepPickProject {
			return m.updateProjectPicker(msg)
		}
		return m.updateBranchInput(msg)
	}

	if m.step == stepEnterBranch {
		var cmd tea.Cmd
		m.branchInput, cmd = m.branchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m createModel) updateProjectPicker(msg tea.KeyMsg) (createModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			m.selectedPath = m.filtered[m.cursor]
			m.step = stepEnterBranch
			m.branchInput.Focus()
			return m, textinput.Blink
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *createModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.projects
	} else {
		m.filtered = nil
		lower := strings.ToLower(m.filter)
		for _, p := range m.projects {
			if strings.Contains(strings.ToLower(filepath.Base(p)), lower) {
				m.filtered = append(m.filtered, p)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m createModel) updateBranchInput(msg tea.KeyMsg) (createModel, tea.Cmd) {
	if msg.String() == "enter" {
		branch := strings.TrimSpace(m.branchInput.Value())
		if branch != "" {
			return m, spawnTabCmd(m.selectedPath, branch)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.branchInput, cmd = m.branchInput.Update(msg)
	return m, cmd
}

func spawnTabCmd(projectDir, branch string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("wezterm", "cli", "spawn", "--cwd", projectDir).Output()
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to create tab: %w", err)}
		}

		paneID := strings.TrimSpace(string(out))

		cmd := exec.Command("wezterm", "cli", "send-text", "--pane-id", paneID, "--no-paste")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("ww %s\n", branch))
		if err := cmd.Run(); err != nil {
			return errMsg{err: fmt.Errorf("tab created but failed to run ww: %w", err)}
		}

		return tabCreatedMsg{}
	}
}

func (m createModel) View() string {
	var b strings.Builder

	if m.step == stepPickProject {
		b.WriteString(modalTitleStyle.Render("Select Project"))
		b.WriteString("\n\n")

		// Filter input
		if m.filter != "" {
			b.WriteString(filterMatchStyle.Render("> " + m.filter))
		} else {
			b.WriteString(filterDimStyle.Render("> type to filter..."))
		}
		b.WriteString("\n\n")

		// Project list
		maxShow := min(10, len(m.filtered))
		start := 0
		if m.cursor >= maxShow {
			start = m.cursor - maxShow + 1
		}
		end := min(start+maxShow, len(m.filtered))

		for i := start; i < end; i++ {
			name := filepath.Base(m.filtered[i])
			if i == m.cursor {
				b.WriteString(lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render("▸ "))
				b.WriteString(lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(name))
			} else {
				b.WriteString("  ")
				b.WriteString(filterDimStyle.Render(name))
			}
			b.WriteString("\n")
		}

		if len(m.filtered) == 0 {
			b.WriteString(filterDimStyle.Render("  no matches"))
		} else if m.filter != "" {
			b.WriteString("\n")
			b.WriteString(filterDimStyle.Render(fmt.Sprintf("  %d of %d projects", len(m.filtered), len(m.projects))))
		}
	} else {
		selectedName := filepath.Base(m.selectedPath)
		b.WriteString(modalTitleStyle.Render("New Tab: " + selectedName))
		b.WriteString("\n\n")
		b.WriteString(m.branchInput.View())
	}

	b.WriteString("\n\n")
	if m.step == stepEnterBranch {
		b.WriteString(filterDimStyle.Render("esc to go back"))
	} else {
		b.WriteString(filterDimStyle.Render("esc to cancel"))
	}

	modalWidth := 60
	if m.width > 0 && m.width-10 < modalWidth {
		modalWidth = m.width - 10
	}

	return modalOverlayStyle.Width(modalWidth).Render(b.String())
}
