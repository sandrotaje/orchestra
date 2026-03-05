# orchestra

A terminal dashboard for monitoring your GitHub PRs across multiple repos, built with Go and Bubble Tea.

```
♪ Orchestra
4 tabs · 4 PRs · 1 failing · 1 ready

┃ interop-be-monorepo (4)
│
│   ╭─ PIN-9237-async-exchange-descriptor-fields * ──────────────╮
│   │ AUTHOR PR #3097 → feature/PIN-92…  Ready to merge         │
│   │ feat: add async exchange descriptor fields (PIN-9237)      │
│   │ ████████████ 24/24  ✓ DenisLa…  +976 -14 46f  8h ago      │
│   ╰──────────────────────────────────────────────────────────╯
│
│   ╭─ PIN-9239_add-check-on-publish-descriptor * ──────────────╮
│   │ REVIEWER PR #3099 → PIN-9237-async…  Draft                 │
│   │ feat: add async exchange validation checks (PIN-9239)      │
│   │ ████████████ 24/24  +499 -2 7f  8h ago                     │
│   ╰──────────────────────────────────────────────────────────╯

j/k navigate  l/Enter switch  n new  x remove  o open PR
y copy  u update  g/G top/end  Tab groups  / filter  r refresh  q quit
```

## Requirements

- [gh CLI](https://cli.github.com/) — must be authenticated
- Go 1.21+

## Installation

```bash
git clone https://github.com/sandrotaje/orchestra
cd orchestra
make install
```

This builds the binary and copies it to `~/.local/bin/orchestra`.

## Usage

```bash
orchestra
```

Run it from anywhere — it scans your configured project directories for git worktrees with open PRs.

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `l` / `Enter` | Switch to worktree |
| `n` | Create new worktree |
| `x` | Remove worktree |
| `o` | Open PR in browser |
| `y` | Copy PR URL |
| `u` | Update branch |
| `g` / `G` | Jump to top/bottom |
| `Tab` | Cycle groups |
| `/` | Filter |
| `r` | Refresh |
| `q` | Quit |

## Features

- **Multi-repo dashboard** — monitors PRs across all your projects, grouped by repo
- **Author/Reviewer badge** — each PR shows `AUTHOR` or `REVIEWER` so you instantly know your role
- **PR status at a glance** — Ready to merge, Draft, Checks failing, Changes requested, Merge conflicts, Needs update
- **CI checks bar** — visual progress bar with pass/fail/pending counts
- **Reviewer summary** — shows approvals and change requests inline
- **Diff stats** — additions, deletions, files changed
- **Worktree integration** — switch to, create, and remove git worktrees directly
- **Auto-refresh** — keeps PR data up to date
- **Filter mode** — search across branches and PRs

## How it works

Uses `gh` CLI under the hood to fetch PR data. Scans project directories for git worktrees and aggregates their PR status into a single dashboard. No GitHub token configuration needed if you're already authenticated with `gh auth login`.
