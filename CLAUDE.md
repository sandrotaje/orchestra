# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
make build        # compile to bin/orchestra
make run          # build + run
make install      # build + copy to ~/.local/bin/orchestra
make clean        # remove bin/
```

There are no tests, linter, or formatter configured yet.

## What This Is

Orchestra is a terminal dashboard that aggregates WezTerm tabs across git projects and enriches them with GitHub PR data. It runs as a Bubble Tea TUI inside a dedicated WezTerm tab.

## Architecture

```
cmd/orchestra/main.go          Entry point — flag parsing, creates Aggregator + TUI Model
internal/aggregator/            Data orchestration layer
  aggregator.go                 Polls WezTerm tabs, resolves git branches, fetches PRs concurrently, groups by project
internal/wezterm/               WezTerm CLI wrapper (`wezterm cli list/spawn/kill-pane`)
  client.go                     ListPanes, GroupByTab, parseTitle (extracts project/branch/claude status from tab title)
  types.go                      Pane and Tab structs
internal/github/                GitHub CLI wrapper (`gh pr view/list/checks`)
  client.go                     GetPRForBranch, getChecks, getReviews, UpdatePRBranch
  types.go                      PRInfo, Check, Review, DiffStat types
internal/tui/                   Bubble Tea UI layer
  model.go                      Main Model — Init/Update/View, message types, viewport scrolling
  card.go                       Card rendering per tab (PR status, checks bar, diff stats, reviews)
  create.go                     Multi-step modal for creating new worktree tabs (project picker → branch input → `ww` spawn)
  keys.go                       Keybinding enum + parseKey mapping
  styles.go                     Catppuccin Mocha color palette + all lipgloss styles
```

## Key Design Decisions

- **Data flow**: Aggregator polls every 1s via `tickMsg`. PR data is cached for 5 minutes (`prCacheTTL`). PRs are fetched concurrently per tab via goroutines.
- **Tab identity**: Tabs are identified by parsing WezTerm tab titles (format: `ICON project/branch`) with fallback to CWD-based worktree path detection (`__worktrees` directory pattern).
- **External tools required at runtime**: `wezterm` CLI, `gh` CLI (authenticated), `ww` (worktree helper for creating/removing worktrees), `pbcopy` (macOS clipboard).
- **Self-exclusion**: The aggregator filters out its own tab (`project=orchestra, branch=orchestra`).
- **TUI pattern**: Standard Bubble Tea architecture — `Model.Update` dispatches to sub-handlers (`updateDashboard`, `updateFilter`, `updateCreate`, `updateConfirmRemove`) based on current UI state. View uses manual viewport scrolling (no viewport component).
