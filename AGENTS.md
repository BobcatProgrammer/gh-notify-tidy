# gh-notify-tidy

A GitHub CLI (`gh`) extension for cleaning up and managing GitHub notifications.
Users invoke it as `gh notify-tidy`.

## Project Structure

- `main.go` — thin entrypoint, delegates to `cmd.Execute()`
- `cmd/` — Cobra subcommands: `read`, `done`, `mute`, `stats`, `interactive`
- `internal/github/` — GitHub REST API client and notification logic
- `internal/tui/` — Bubble Tea interactive TUI model and lipgloss styles
- `.github/workflows/` — CI: lint, PR title validation, semantic release, binary compilation

## Commands

- `read` — Mark notifications as read (`PATCH /notifications/threads/{id}`)
- `done` — Delete subscription / "Done" (`DELETE /notifications/threads/{id}/subscription`)
- `mute` — Mute thread (`PUT /notifications/threads/{id}/subscription` with `ignored: true`)
- `stats` — Notification statistics table for last N days with subscription suggestions
- `interactive` — Bubble Tea TUI that walks through cleanup steps

Global flags on all commands: `--repo <owner/repo>`, `--org <org>`, `--host <hostname>`, `--dry-run`

## Code Standards

- Go module: `github.com/BobcatProgrammer/gh-notify-tidy`
- Go version: as specified in `go.mod`
- All Go code must pass `golangci-lint run ./...` with the config in `.golangci.yml`
- Formatting: `gofmt` + `goimports` (enforced by golangci-lint)
- Imports: group as stdlib / external / internal, separated by blank lines
- Errors: wrap with `fmt.Errorf("context: %w", err)`, never discard
- No `log.Fatal` outside of `main.go`; return errors up the call stack

## Key Dependencies

- `github.com/cli/go-gh/v2` — REST client (auto-inherits `gh` auth), terminal utils
- `github.com/spf13/cobra` — CLI subcommand framework
- `github.com/charmbracelet/bubbletea` — interactive TUI
- `github.com/charmbracelet/lipgloss` — TUI styling
- `github.com/charmbracelet/bubbles` — list, spinner, table components

## Authentication

Use `api.DefaultRESTClient()` from `go-gh` — it automatically picks up the user's
`gh auth` token. For GHES, construct with `api.ClientOptions{Host: host}` using the
`--host` flag value. Never hardcode tokens.

## GitHub Notifications API

- List: `GET notifications?all=true&per_page=50` (paginate via `Link` header `rel="next"`)
- Mark read: `PATCH notifications/threads/{id}` → 205
- Done (archive): `DELETE notifications/threads/{id}/subscription` → 204
- Mute: `PUT notifications/threads/{id}/subscription` body `{"ignored":true}` → 200
- Check PR state: `GET repos/{owner}/{repo}/pulls/{number}` → check `.state` and `.merged`

## PR & Release Workflow

- All changes go through PRs targeting `main`
- PR titles **must** follow Conventional Commits: `feat:`, `fix:`, `chore:`, `docs:`,
  `refactor:`, `test:`, `ci:`, `perf:`
- PRs are squash-merged; the PR title becomes the squash commit message
- Merging to `main` triggers semantic-release which creates the version tag and GitHub Release
- The version tag triggers `gh-extension-precompile` to build and upload binaries
- Breaking changes: use `feat!:` or `fix!:` in the PR title

## Code Quality

- Pre-commit hooks: `trailing-whitespace`, `end-of-file-fixer`, `check-yaml`,
  `check-merge-conflict`, `mixed-line-ending`, `golangci-lint`, `markdownlint-cli2`
- CI lint workflow runs on every PR and push to `main`
- Never commit code that fails `golangci-lint run ./...`
- Markdown files must pass `markdownlint-cli2` with `.markdownlint.yaml` config

## Testing

- Unit tests use the standard `testing` package
- Table-driven tests preferred
- Test files follow `_test.go` convention in the same package (white-box) or `_test`
  suffix package (black-box)
- Mock the GitHub API client via interface — do not make real API calls in tests
