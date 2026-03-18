package github

import (
	"testing"
	"time"
)

// ── nextLink / splitLinkHeader / parseLinkEntry ───────────────────────────────

func TestNextLink(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "empty header",
			header: "",
			want:   "",
		},
		{
			name:   "single next rel",
			header: `<https://api.github.com/notifications?page=2>; rel="next"`,
			want:   "https://api.github.com/notifications?page=2",
		},
		{
			name:   "no next rel — only last",
			header: `<https://api.github.com/notifications?page=1>; rel="last"`,
			want:   "",
		},
		{
			name: "multi-entry — next and last",
			header: `<https://api.github.com/notifications?page=2>; rel="next", ` +
				`<https://api.github.com/notifications?page=5>; rel="last"`,
			want: "https://api.github.com/notifications?page=2",
		},
		{
			name: "multi-entry — prev and next",
			header: `<https://api.github.com/notifications?page=1>; rel="prev", ` +
				`<https://api.github.com/notifications?page=3>; rel="next"`,
			want: "https://api.github.com/notifications?page=3",
		},
		{
			name:   "URL contains comma inside angle brackets",
			header: `<https://example.com/path?a=1,b=2>; rel="next"`,
			want:   "https://example.com/path?a=1,b=2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nextLink(tc.header)
			if got != tc.want {
				t.Errorf("nextLink(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestSplitLinkHeader(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single entry",
			input: `<https://example.com/page=2>; rel="next"`,
			want:  []string{`<https://example.com/page=2>; rel="next"`},
		},
		{
			name:  "two entries",
			input: `<https://example.com/page=2>; rel="next", <https://example.com/page=5>; rel="last"`,
			want: []string{
				`<https://example.com/page=2>; rel="next"`,
				` <https://example.com/page=5>; rel="last"`,
			},
		},
		{
			name:  "comma inside angle brackets not treated as separator",
			input: `<https://example.com/path?a=1,b=2>; rel="next"`,
			want:  []string{`<https://example.com/path?a=1,b=2>; rel="next"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitLinkHeader(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("splitLinkHeader(%q) returned %d parts, want %d: %v", tc.input, len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("part %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseLinkEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		wantURL string
		wantRel string
	}{
		{
			name:    "valid next entry",
			entry:   `<https://api.github.com/notifications?page=2>; rel="next"`,
			wantURL: "https://api.github.com/notifications?page=2",
			wantRel: "next",
		},
		{
			name:    "valid last entry",
			entry:   `<https://api.github.com/notifications?page=5>; rel="last"`,
			wantURL: "https://api.github.com/notifications?page=5",
			wantRel: "last",
		},
		{
			name:    "missing opening angle bracket",
			entry:   `https://api.github.com/notifications?page=2>; rel="next"`,
			wantURL: "",
			wantRel: "",
		},
		{
			name:    "missing closing angle bracket",
			entry:   `<https://api.github.com/notifications?page=2; rel="next"`,
			wantURL: "",
			wantRel: "",
		},
		{
			name:    "entry with leading whitespace",
			entry:   ` <https://api.github.com/notifications?page=3>; rel="next"`,
			wantURL: "https://api.github.com/notifications?page=3",
			wantRel: "next",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, gotRel := parseLinkEntry(tc.entry)
			if gotURL != tc.wantURL {
				t.Errorf("parseLinkEntry url: got %q, want %q", gotURL, tc.wantURL)
			}
			if gotRel != tc.wantRel {
				t.Errorf("parseLinkEntry rel: got %q, want %q", gotRel, tc.wantRel)
			}
		})
	}
}

// ── Filter.matches ────────────────────────────────────────────────────────────

func makeThread(repo, org string, unread bool, age time.Duration) Thread {
	t := Thread{}
	t.Repository.FullName = repo
	t.Repository.Owner.Login = org
	t.Unread = unread
	t.UpdatedAt = time.Now().Add(-age)
	return t
}

func TestFilterMatches(t *testing.T) {
	recentThread := makeThread("org/repo", "org", true, 1*time.Hour)
	oldThread := makeThread("org/repo", "org", false, 48*time.Hour)
	otherRepoThread := makeThread("org/other", "org", true, 1*time.Hour)
	otherOrgThread := makeThread("other/repo", "other", true, 1*time.Hour)

	tests := []struct {
		name   string
		filter Filter
		thread Thread
		want   bool
	}{
		{
			name:   "empty filter matches everything",
			filter: Filter{},
			thread: recentThread,
			want:   true,
		},
		{
			name:   "repo filter matches correct repo",
			filter: Filter{Repo: "org/repo"},
			thread: recentThread,
			want:   true,
		},
		{
			name:   "repo filter excludes other repo",
			filter: Filter{Repo: "org/repo"},
			thread: otherRepoThread,
			want:   false,
		},
		{
			name:   "org filter matches correct org",
			filter: Filter{Org: "org"},
			thread: recentThread,
			want:   true,
		},
		{
			name:   "org filter excludes other org",
			filter: Filter{Org: "org"},
			thread: otherOrgThread,
			want:   false,
		},
		{
			name:   "older-than excludes recent thread",
			filter: Filter{OlderThan: 24 * time.Hour},
			thread: recentThread,
			want:   false,
		},
		{
			name:   "older-than includes old thread",
			filter: Filter{OlderThan: 24 * time.Hour},
			thread: oldThread,
			want:   true,
		},
		{
			name:   "only-read excludes unread thread",
			filter: Filter{OnlyRead: true},
			thread: recentThread,
			want:   false,
		},
		{
			name:   "only-read includes read thread",
			filter: Filter{OnlyRead: true},
			thread: oldThread,
			want:   true,
		},
		{
			name:   "combined repo and only-read — match",
			filter: Filter{Repo: "org/repo", OnlyRead: true},
			thread: oldThread,
			want:   true,
		},
		{
			name:   "combined repo and only-read — wrong repo",
			filter: Filter{Repo: "org/repo", OnlyRead: true},
			thread: makeThread("org/other", "org", false, 48*time.Hour),
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.filter.matches(tc.thread)
			if got != tc.want {
				t.Errorf("Filter.matches() = %v, want %v", got, tc.want)
			}
		})
	}
}
