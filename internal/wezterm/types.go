package wezterm

// Pane represents a single pane from `wezterm cli list --format json`.
type Pane struct {
	WindowID  int    `json:"window_id"`
	TabID     int    `json:"tab_id"`
	PaneID    int    `json:"pane_id"`
	Workspace string `json:"workspace"`
	Title     string `json:"title"`
	TabTitle  string `json:"tab_title"`
	CWD       string `json:"cwd"`
	IsActive  bool   `json:"is_active"`
}

// Tab groups all panes sharing the same tab_id.
type Tab struct {
	TabID        int
	PaneIDs      []int
	Title        string
	CWD          string
	IsActive     bool
	Project      string // parsed from title or cwd
	Branch       string // parsed from title or cwd
	ClaudeStatus string // "working" | "done" | "notification" | ""
}
