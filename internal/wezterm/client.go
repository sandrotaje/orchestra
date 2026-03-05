package wezterm

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ListPanes returns all panes from wezterm cli list.
func ListPanes() ([]Pane, error) {
	out, err := run("cli", "list", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("wezterm cli list: %w", err)
	}
	var panes []Pane
	if err := json.Unmarshal([]byte(out), &panes); err != nil {
		return nil, fmt.Errorf("parse pane list: %w", err)
	}
	return panes, nil
}

// ActivateTab switches to the given tab.
func ActivateTab(tabID int) error {
	_, err := run("cli", "activate-tab", "--tab-id", strconv.Itoa(tabID))
	return err
}

// GroupByTab groups panes into Tabs and parses project/branch from titles.
func GroupByTab(panes []Pane) []Tab {
	tabMap := make(map[int]*Tab)
	var order []int

	for _, p := range panes {
		t, ok := tabMap[p.TabID]
		if !ok {
			t = &Tab{
				TabID: p.TabID,
				Title: p.TabTitle,
				CWD:   p.CWD,
			}
			tabMap[p.TabID] = t
			order = append(order, p.TabID)
		}
		t.PaneIDs = append(t.PaneIDs, p.PaneID)
		if p.IsActive {
			t.IsActive = true
		}
		// Prefer tab_title; fallback to first pane's title
		if t.Title == "" {
			t.Title = p.Title
		}
		// Keep cwd from the first pane (usually the Claude pane)
		if t.CWD == "" {
			t.CWD = p.CWD
		}
	}

	tabs := make([]Tab, 0, len(order))
	for _, id := range order {
		t := tabMap[id]
		t.Project, t.Branch, t.ClaudeStatus = parseTitle(t.Title, t.CWD)
		tabs = append(tabs, *t)
	}
	return tabs
}

// parseTitle extracts project, branch, and claude status from a tab title.
// Title format from wez-tab-status.sh: "ICON project/branch"
// Fallback: parse cwd for __worktrees pattern.
func parseTitle(title, cwd string) (project, branch, claudeStatus string) {
	title = strings.TrimSpace(title)

	if title != "" {
		// Check for status icon prefix
		for icon, status := range map[string]string{
			"⏳": "working",
			"✅": "done",
			"💬": "notification",
		} {
			if strings.HasPrefix(title, icon) {
				claudeStatus = status
				title = strings.TrimSpace(strings.TrimPrefix(title, icon))
				break
			}
		}

		// Parse "project/branch" from remaining title
		if idx := strings.Index(title, "/"); idx > 0 {
			project = title[:idx]
			branch = title[idx+1:]
			return
		}

		// No slash — title is just a name (could be branch or project)
		branch = title
	}

	// Fallback: parse CWD for worktree pattern
	if cwd != "" {
		cwd = strings.TrimPrefix(cwd, "file://")
		// Strip hostname from file:// URI (e.g. file://hostname/path)
		if idx := strings.Index(cwd, "/Users"); idx > 0 {
			cwd = cwd[idx:]
		}

		dir := filepath.Base(filepath.Dir(cwd))
		if strings.HasSuffix(dir, "__worktrees") {
			project = strings.TrimSuffix(dir, "__worktrees")
			branch = filepath.Base(cwd)
			return
		}
		// Not a worktree — use directory name as project
		if project == "" {
			project = filepath.Base(cwd)
		}
		if branch == "" {
			branch = project
		}
	}

	return
}

func run(args ...string) (string, error) {
	cmd := exec.Command("wezterm", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
