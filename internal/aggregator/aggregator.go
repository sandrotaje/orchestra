package aggregator

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"orchestra/internal/github"
	"orchestra/internal/wezterm"
)

// DashboardTab holds the combined state for a single WezTerm tab.
type DashboardTab struct {
	TabID        int
	Project      string
	Branch       string // display name (may be filesystem-safe)
	GitBranch    string // real git branch name (for gh lookups)
	ClaudeStatus string
	IsActive     bool
	PR           *github.PRInfo
	PaneIDs      []int
	RepoPath     string
	WorkDir      string // cleaned CWD from WezTerm tab
}

// ProjectGroup holds tabs grouped under a project name.
type ProjectGroup struct {
	Name string
	Tabs []DashboardTab
}

type prCacheEntry struct {
	pr        *github.PRInfo
	fetchedAt time.Time
}

// Aggregator orchestrates data fetching, caching, and merging.
type Aggregator struct {
	projectsDir string

	mu      sync.Mutex
	prCache map[string]*prCacheEntry
	repoDirs map[string]string // projectName → absolute repo path
}

const prCacheTTL = 1 * time.Minute

func New(projectsDir string) *Aggregator {
	return &Aggregator{
		projectsDir: projectsDir,
		prCache:     make(map[string]*prCacheEntry),
		repoDirs:    make(map[string]string),
	}
}

// Refresh polls WezTerm for tabs and returns them grouped by project.
func (a *Aggregator) Refresh() []ProjectGroup {
	panes, err := wezterm.ListPanes()
	if err != nil {
		return nil
	}

	tabs := wezterm.GroupByTab(panes)

	// Ensure repo dirs are scanned
	a.mu.Lock()
	if len(a.repoDirs) == 0 {
		a.scanRepoDirs()
	}
	a.mu.Unlock()

	// Build dashboard tabs (filter out orchestra's own tab)
	dtabs := make([]DashboardTab, 0, len(tabs))
	for _, t := range tabs {
		if t.Project == "orchestra" && t.Branch == "orchestra" {
			continue
		}
		dt := DashboardTab{
			TabID:        t.TabID,
			Project:      t.Project,
			Branch:       t.Branch,
			ClaudeStatus: t.ClaudeStatus,
			IsActive:     t.IsActive,
			PaneIDs:      t.PaneIDs,
			WorkDir:      cleanCWD(t.CWD),
		}

		// Resolve repo path
		dt.RepoPath = a.resolveRepoPath(t.Project, t.CWD)

		// Resolve real git branch name from worktree CWD
		dt.GitBranch = resolveGitBranch(t.CWD)
		if dt.GitBranch == "" {
			dt.GitBranch = dt.Branch // fallback to display name
		}

		dtabs = append(dtabs, dt)
	}

	// Fetch PRs concurrently
	var wg sync.WaitGroup
	for i := range dtabs {
		dt := &dtabs[i]
		if dt.RepoPath != "" && dt.GitBranch != "" && dt.GitBranch != dt.Project {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dt.PR = a.getPR(dt.RepoPath, dt.GitBranch)
			}()
		}
	}
	wg.Wait()

	return groupByProject(dtabs)
}

// ClearPRCache forces re-fetch of all PR data on next Refresh.
func (a *Aggregator) ClearPRCache() {
	a.mu.Lock()
	a.prCache = make(map[string]*prCacheEntry)
	a.mu.Unlock()
}

// PRCacheTTL returns the cache TTL duration for display purposes.
func PRCacheTTL() time.Duration {
	return prCacheTTL
}

// OldestPRFetch returns the oldest fetchedAt time in the cache.
// Returns zero time if cache is empty.
func (a *Aggregator) OldestPRFetch() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()

	var oldest time.Time
	for _, entry := range a.prCache {
		if oldest.IsZero() || entry.fetchedAt.Before(oldest) {
			oldest = entry.fetchedAt
		}
	}
	return oldest
}

func (a *Aggregator) getPR(repoDir, branch string) *github.PRInfo {
	key := repoDir + ":" + branch

	a.mu.Lock()
	if entry, ok := a.prCache[key]; ok && time.Since(entry.fetchedAt) < prCacheTTL {
		a.mu.Unlock()
		return entry.pr
	}
	a.mu.Unlock()

	pr, _ := github.GetPRForBranch(repoDir, branch)

	a.mu.Lock()
	a.prCache[key] = &prCacheEntry{pr: pr, fetchedAt: time.Now()}
	a.mu.Unlock()

	return pr
}

// resolveGitBranch runs `git branch --show-current` in the tab's CWD to get
// the real branch name (which may differ from the filesystem-safe directory name).
func resolveGitBranch(cwd string) string {
	cwd = cleanCWD(cwd)
	if cwd == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", cwd, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// cleanCWD strips the file:// URI scheme and hostname from WezTerm CWDs.
func cleanCWD(cwd string) string {
	if cwd == "" {
		return ""
	}
	cwd = strings.TrimPrefix(cwd, "file://")
	if idx := strings.Index(cwd, "/Users"); idx > 0 {
		cwd = cwd[idx:]
	}
	return cwd
}

func (a *Aggregator) resolveRepoPath(project, cwd string) string {
	// Try direct lookup by project name
	a.mu.Lock()
	if p, ok := a.repoDirs[project]; ok {
		a.mu.Unlock()
		return p
	}
	a.mu.Unlock()

	// Fallback: walk up from cwd to find .git
	cwd = cleanCWD(cwd)
	if cwd != "" {
		dir := cwd
		for {
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return ""
}

func (a *Aggregator) scanRepoDirs() {
	entries, err := os.ReadDir(a.projectsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		full := filepath.Join(a.projectsDir, e.Name())
		if _, err := os.Stat(filepath.Join(full, ".git")); err == nil {
			a.repoDirs[e.Name()] = full
		}
	}
}

// ListProjects returns project directories with .git repos for the create modal.
func (a *Aggregator) ListProjects() []string {
	a.mu.Lock()
	if len(a.repoDirs) == 0 {
		a.scanRepoDirs()
	}
	a.mu.Unlock()

	a.mu.Lock()
	defer a.mu.Unlock()

	projects := make([]string, 0, len(a.repoDirs))
	for _, path := range a.repoDirs {
		projects = append(projects, path)
	}
	sort.Strings(projects)
	return projects
}

func groupByProject(tabs []DashboardTab) []ProjectGroup {
	groups := make(map[string][]DashboardTab)
	var order []string

	for _, t := range tabs {
		name := t.Project
		if name == "" {
			name = "other"
		}
		if _, ok := groups[name]; !ok {
			order = append(order, name)
		}
		groups[name] = append(groups[name], t)
	}

	// Sort group names alphabetically, but keep "other" at the bottom
	sort.SliceStable(order, func(i, j int) bool {
		if order[i] == "other" {
			return false
		}
		if order[j] == "other" {
			return true
		}
		return order[i] < order[j]
	})

	result := make([]ProjectGroup, len(order))
	for i, name := range order {
		tabsInGroup := groups[name]
		sort.Slice(tabsInGroup, func(a, b int) bool {
			return tabsInGroup[a].Branch < tabsInGroup[b].Branch
		})
		result[i] = ProjectGroup{Name: name, Tabs: tabsInGroup}
	}

	return result
}
