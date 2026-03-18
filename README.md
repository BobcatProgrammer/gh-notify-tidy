# gh-notify-tidy

A [GitHub CLI](https://cli.github.com/) extension for cleaning up and managing GitHub notifications.
It focuses on automatically identifying **stale notifications** — ones that no longer require action
(merged/closed PRs, already-approved PRs, closed issues) — and archiving them in bulk.

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

| Command | Description |
| --- | --- |
| `pr` | Find stale PR and issue notifications (closed/merged PRs, closed issues, already-approved PRs) |
| `ci` | Find stale CI activity notifications *(stub — see [#12][i12])* |
| `all` | Run all staleness checks in one pass |
| `old` | Find notifications older than N days |
| `interactive` | Guided batch-cleanup TUI (recommended) |
| `stats` | Per-repository notification breakdown with suggestions |
| `read` | Mark notifications as read |
| `done` | Archive notifications (delete subscription) |
| `mute` | Mute notification threads |

### Global flags

| Flag | Description |
| --- | --- |
| `--repo <owner/repo>` | Filter to a single repository |
| `--org <org>` | Filter to an organisation |
| `--host <hostname>` | GitHub hostname (for GitHub Enterprise Server) |
| `--dry-run` | Print what would be done without making changes |

---

### `gh notify-tidy pr`

Identify notifications that no longer require action because the underlying pull request or issue
has been closed, merged, or already reviewed and approved by a colleague.

By default matching notifications are printed. Use `--done`, `--read`, or `--mute` to act on them.

```bash
gh notify-tidy pr [--done | --read | --mute] [--repo owner/repo] [--org myorg] [--dry-run]
```

**Staleness rules checked:**

| Reason | Subject | Stale when |
| --- | --- | --- |
| `review_requested`, `assign`, `mention`, `subscribed`, `team_mention` | `PullRequest` | PR closed or merged |
| `review_requested` | `PullRequest` | PR open but ≥1 approved review from someone other than you |
| `assign`, `mention`, `subscribed`, `team_mention` | `Issue` | Issue closed |

---

### `gh notify-tidy ci`

Identify `ci_activity` notifications where the check run has since passed.
**Not yet implemented** — track progress in [issue #12][i12].

---

### `gh notify-tidy all`

Run all available staleness checks (`pr` + future `ci`) in one pass.

```bash
gh notify-tidy all [--done | --read | --mute] [--older-than N] [--repo owner/repo] [--org myorg] [--dry-run]
```

| Flag | Description |
| --- | --- |
| `--older-than N` | Also include notifications not updated in N days |

---

### `gh notify-tidy old`

Find notifications that have not been updated in more than N days.

```bash
gh notify-tidy old --older-than N [--done | --read | --mute] [--repo owner/repo] [--org myorg] [--dry-run]
```

| Flag | Description |
| --- | --- |
| `--older-than N` | Age threshold in days (required) |

---

### `gh notify-tidy interactive`

Walk through your notifications in a Bubble Tea TUI. Notifications are grouped by
`(repository, stale reason)` and you assign a bulk action to each group before applying.

```bash
gh notify-tidy interactive [--repo owner/repo] [--org myorg] [--dry-run]
```

**Steps:**

1. **Loading** — fetch notifications and check staleness concurrently
2. **Statistics** — per-repo breakdown with subscription suggestions
3. **Batch groups** — assign an action to each `(repo, reason)` group
4. **Confirm** — review the full list before applying
5. **Applying** — execute selected actions
6. **Done** — summary

**Keys in Batch groups step:**

| Key | Action |
| --- | --- |
| `d` | Done (archive) — default |
| `r` | Mark as read |
| `m` | Mute thread |
| `s` | Skip this group |
| `enter` | Confirm current group and move on |
| `q` | Quit |

---

### `gh notify-tidy done`

Delete the subscription for matching notifications ("Done" / archive).

```bash
gh notify-tidy done [--auto] [--older-than N] [--read] [--dry-run]
```

| Flag | Description |
| --- | --- |
| `--auto` | Run full staleness check and archive all stale threads automatically |
| `--older-than N` | Only include notifications not updated in N days |
| `--read` | Only include already-read notifications |

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

---

### `gh notify-tidy mute`

Set `ignored=true` on the subscription for matching threads.
Muted threads will no longer generate notifications.

```bash
gh notify-tidy mute [--older-than N] [--repo owner/repo] [--org myorg] [--dry-run]
```

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

[i12]: https://github.com/BobcatProgrammer/gh-notify-tidy/issues/12
