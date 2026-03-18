package github

import (
	"testing"
	"time"
)

// ── stripAPIBase ──────────────────────────────────────────────────────────────

func TestStripAPIBase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "github.com full URL",
			input: "https://api.github.com/repos/owner/repo/pulls/123",
			want:  "repos/owner/repo/pulls/123",
		},
		{
			name:  "GHES URL with api/v3 prefix",
			input: "https://github.example.com/api/v3/repos/owner/repo/pulls/42",
			want:  "repos/owner/repo/pulls/42",
		},
		{
			name:  "no scheme — first path segment treated as host and stripped",
			input: "repos/owner/repo/pulls/1",
			want:  "owner/repo/pulls/1",
		},
		{
			name:  "http scheme",
			input: "http://api.github.com/repos/owner/repo/issues/5",
			want:  "repos/owner/repo/issues/5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripAPIBase(tc.input)
			if got != tc.want {
				t.Errorf("stripAPIBase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── ComputeStats ──────────────────────────────────────────────────────────────

// threadFor creates a Thread for the given repo and reason.
// The owner login is derived from the repo's org portion.
func threadFor(repo, reason string, unread bool) Thread {
	th := Thread{}
	th.Repository.FullName = repo
	th.Repository.Owner.Login = "org"
	th.Reason = reason
	th.Unread = unread
	th.UpdatedAt = time.Now()
	return th
}

func TestComputeStats_SingleRepo(t *testing.T) {
	threads := []Thread{
		threadFor("org/repo", "subscribed", true),
		threadFor("org/repo", "subscribed", false),
		threadFor("org/repo", "review_requested", true),
	}

	stats := ComputeStats(threads)

	if len(stats) != 1 {
		t.Fatalf("expected 1 stats entry, got %d", len(stats))
	}

	s := stats[0]
	if s.Repo != "org/repo" {
		t.Errorf("Repo = %q, want %q", s.Repo, "org/repo")
	}
	if s.Total != 3 {
		t.Errorf("Total = %d, want 3", s.Total)
	}
	if s.Unread != 2 {
		t.Errorf("Unread = %d, want 2", s.Unread)
	}
	if s.ByReason["subscribed"] != 2 {
		t.Errorf("ByReason[subscribed] = %d, want 2", s.ByReason["subscribed"])
	}
	if s.ByReason["review_requested"] != 1 {
		t.Errorf("ByReason[review_requested] = %d, want 1", s.ByReason["review_requested"])
	}
}

func TestComputeStats_MultiRepoSortedByTotal(t *testing.T) {
	threads := []Thread{
		threadFor("org/small", "subscribed", false),
		threadFor("org/large", "subscribed", true),
		threadFor("org/large", "subscribed", true),
		threadFor("org/large", "subscribed", true),
	}

	stats := ComputeStats(threads)

	if len(stats) != 2 {
		t.Fatalf("expected 2 stats entries, got %d", len(stats))
	}
	if stats[0].Repo != "org/large" {
		t.Errorf("expected org/large first (highest total), got %q", stats[0].Repo)
	}
	if stats[0].Total != 3 {
		t.Errorf("org/large Total = %d, want 3", stats[0].Total)
	}
	if stats[1].Total != 1 {
		t.Errorf("org/small Total = %d, want 1", stats[1].Total)
	}
}

func TestComputeStats_Empty(t *testing.T) {
	stats := ComputeStats(nil)
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}

// ── suggest (exercised through ComputeStats) ──────────────────────────────────

// makeSuggestionThreads builds a slice of threads for a single repo "r"
// with the requested distribution of reasons and unread count.
func makeSuggestionThreads(total, subscribed, reviewReq, ciActivity, unread int) []Thread {
	const repo = "org/r"

	var threads []Thread
	for i := 0; i < subscribed; i++ {
		threads = append(threads, threadFor(repo, "subscribed", i < unread))
	}
	for i := 0; i < reviewReq; i++ {
		threads = append(threads, threadFor(repo, "review_requested", false))
	}
	for i := 0; i < ciActivity; i++ {
		threads = append(threads, threadFor(repo, "ci_activity", false))
	}
	// pad to total with neutral reason
	other := total - subscribed - reviewReq - ciActivity
	for i := 0; i < other; i++ {
		threads = append(threads, threadFor(repo, "mention", false))
	}
	return threads
}

func TestSuggest(t *testing.T) {
	tests := []struct {
		name       string
		threads    []Thread
		wantSubstr string // empty means no suggestion expected
	}{
		{
			name:       "high subscribed volume → unwatch",
			threads:    makeSuggestionThreads(21, 15, 0, 0, 0),
			wantSubstr: "unwatch",
		},
		{
			name:       "high review_requested volume → mute",
			threads:    makeSuggestionThreads(11, 0, 9, 0, 0),
			wantSubstr: "mute threads after reviewing",
		},
		{
			name:       "mostly ci_activity → disable CI",
			threads:    makeSuggestionThreads(4, 0, 0, 3, 0),
			wantSubstr: "CI",
		},
		{
			name:       "all read high volume → run done",
			threads:    makeSuggestionThreads(10, 0, 0, 0, 0),
			wantSubstr: "done",
		},
		{
			name:       "low volume — no suggestion",
			threads:    makeSuggestionThreads(3, 2, 0, 0, 1),
			wantSubstr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stats := ComputeStats(tc.threads)
			if len(stats) == 0 {
				if tc.wantSubstr != "" {
					t.Fatalf("no stats returned, wanted suggestion containing %q", tc.wantSubstr)
				}
				return
			}

			suggestion := stats[0].Suggestion
			if tc.wantSubstr == "" {
				if suggestion != "" {
					t.Errorf("expected no suggestion, got %q", suggestion)
				}
				return
			}

			if !containsStr(suggestion, tc.wantSubstr) {
				t.Errorf("suggestion = %q, want it to contain %q", suggestion, tc.wantSubstr)
			}
		})
	}
}

// containsStr is a simple substring check to avoid importing strings in tests.
func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
