# gh-notify-tidy

A [GitHub CLI](https://cli.github.com/) extension for cleaning up and managing GitHub notifications.

## Installation

```bash
gh extension install BobcatProgrammer/gh-notify-tidy
```

Requires `gh` 2.0+ with an active `gh auth login` session.

## Usage

```text
gh notify-tidy <command> [flags]
```

### Commands

| Command | Alias | Description |
| --- | --- | --- |
| `interactive` | `i` | Guided step-by-step cleanup TUI (recommended) |
| `stats` | | Per-repository notification breakdown with suggestions |
| `read` | | Mark notifications as read |
| `done` | | Archive notifications (delete subscription) |
| `mute` | | Mute notification threads |

### Global flags

| Flag | Description |
| --- | --- |
| `--repo <owner/repo>` | Filter to a single repository |
| `--org <org>` | Filter to an organisation |
| `--host <hostname>` | GitHub hostname (for GitHub Enterprise Server) |
| `--dry-run` | Print what would be done without making changes |

---

### `gh notify-tidy interactive`

Walk through your notifications step-by-step in a Bubble Tea TUI.

```bash
gh notify-tidy interactive [--old-days N]
```

**Steps:**

1. **Statistics** — per-repo breakdown with subscription suggestions
2. **Old notifications** — triage notifications older than N days (default: 30)
3. **Closed/merged PR notifications** — triage read PR threads
4. **Already-read notifications** — triage all other read threads
5. **Confirm** — review and apply selected actions

**Keys:**

| Key | Action |
| --- | --- |
| `r` | Mark as read |
| `d` | Done (archive) |
| `m` | Mute thread |
| `s` | Skip |
| `enter` / `n` | Next step |
| `q` | Quit |

---

### `gh notify-tidy stats`

Display a per-repository notification breakdown with subscription suggestions.

```bash
gh notify-tidy stats [--repo owner/repo] [--org myorg]
```

---

### `gh notify-tidy read`

Mark matching notifications as read.

```bash
gh notify-tidy read [--older-than N] [--repo owner/repo] [--org myorg] [--dry-run]
```

| Flag | Description |
| --- | --- |
| `--older-than N` | Only include notifications not updated in N days |

---

### `gh notify-tidy done`

Delete the subscription for matching notifications ("Done" / archive).

```bash
gh notify-tidy done [--older-than N] [--read] [--closed-prs] [--dry-run]
```

| Flag | Description |
| --- | --- |
| `--older-than N` | Only include notifications not updated in N days |
| `--read` | Only include already-read notifications |
| `--closed-prs` | Only include read notifications for closed/merged PRs |

---

### `gh notify-tidy mute`

Set `ignored=true` on the subscription for matching threads. Muted threads will no longer generate notifications.

```bash
gh notify-tidy mute [--older-than N] [--repo owner/repo] [--org myorg] [--dry-run]
```

| Flag | Description |
| --- | --- |
| `--older-than N` | Only include notifications not updated in N days |

---

## GitHub Enterprise Server

Pass `--host` to target a GHES instance:

```bash
gh notify-tidy stats --host github.example.com
```

The extension uses the same token that `gh auth login --hostname github.example.com` stores.

## Contributing

1. Fork and clone the repository.
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make changes; ensure `golangci-lint run ./...` and `go test ./...` pass.
4. Open a PR with a title following [Conventional Commits](https://www.conventionalcommits.org/):
   `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `ci:`, `perf:`
5. PRs are squash-merged; the PR title becomes the commit message that drives
   [semantic-release](https://github.com/semantic-release/semantic-release).

## License

[MIT](LICENSE)
