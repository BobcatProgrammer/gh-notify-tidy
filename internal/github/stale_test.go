package github

import (
	"testing"
)

// ── hasApprovalFromOther ──────────────────────────────────────────────────────

func TestHasApprovalFromOther(t *testing.T) {
	tests := []struct {
		name        string
		reviews     []PRReview
		viewerLogin string
		want        bool
	}{
		{
			name:        "empty reviews → false",
			reviews:     nil,
			viewerLogin: "alice",
			want:        false,
		},
		{
			name: "only viewer approved → false",
			reviews: []PRReview{
				{State: "APPROVED", User: struct {
					Login string `json:"login"`
				}{Login: "alice"}},
			},
			viewerLogin: "alice",
			want:        false,
		},
		{
			name: "another user approved → true",
			reviews: []PRReview{
				{State: "APPROVED", User: struct {
					Login string `json:"login"`
				}{Login: "bob"}},
			},
			viewerLogin: "alice",
			want:        true,
		},
		{
			name: "PENDING review ignored",
			reviews: []PRReview{
				{State: "PENDING", User: struct {
					Login string `json:"login"`
				}{Login: "bob"}},
			},
			viewerLogin: "alice",
			want:        false,
		},
		{
			name: "latest review wins — CHANGES_REQUESTED after APPROVED",
			reviews: []PRReview{
				{State: "APPROVED", User: struct {
					Login string `json:"login"`
				}{Login: "bob"}},
				{State: "CHANGES_REQUESTED", User: struct {
					Login string `json:"login"`
				}{Login: "bob"}},
			},
			viewerLogin: "alice",
			want:        false,
		},
		{
			name: "latest review wins — APPROVED after CHANGES_REQUESTED",
			reviews: []PRReview{
				{State: "CHANGES_REQUESTED", User: struct {
					Login string `json:"login"`
				}{Login: "bob"}},
				{State: "APPROVED", User: struct {
					Login string `json:"login"`
				}{Login: "bob"}},
			},
			viewerLogin: "alice",
			want:        true,
		},
		{
			name: "multiple reviewers, one approved",
			reviews: []PRReview{
				{State: "CHANGES_REQUESTED", User: struct {
					Login string `json:"login"`
				}{Login: "carol"}},
				{State: "APPROVED", User: struct {
					Login string `json:"login"`
				}{Login: "bob"}},
			},
			viewerLogin: "alice",
			want:        true,
		},
		{
			name: "empty login skipped",
			reviews: []PRReview{
				{State: "APPROVED", User: struct {
					Login string `json:"login"`
				}{Login: ""}},
			},
			viewerLogin: "alice",
			want:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasApprovalFromOther(tc.reviews, tc.viewerLogin)
			if got != tc.want {
				t.Errorf("hasApprovalFromOther() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ── StaleOnly ─────────────────────────────────────────────────────────────────

func TestStaleOnly(t *testing.T) {
	threads := []StaleThread{
		{Thread: Thread{ID: "1"}, StaleReason: StaleReasonPRMergedClosed},
		{Thread: Thread{ID: "2"}, StaleReason: ""},
		{Thread: Thread{ID: "3"}, StaleReason: StaleReasonIssueClosed},
		{Thread: Thread{ID: "4"}, StaleReason: ""},
	}

	got := StaleOnly(threads)

	if len(got) != 2 {
		t.Fatalf("StaleOnly() returned %d items, want 2", len(got))
	}
	if got[0].Thread.ID != "1" {
		t.Errorf("first stale ID = %q, want %q", got[0].Thread.ID, "1")
	}
	if got[1].Thread.ID != "3" {
		t.Errorf("second stale ID = %q, want %q", got[1].Thread.ID, "3")
	}
}

func TestStaleOnly_Empty(t *testing.T) {
	got := StaleOnly(nil)
	if len(got) != 0 {
		t.Errorf("StaleOnly(nil) returned %d items, want 0", len(got))
	}
}

func TestStaleOnly_NoneStale(t *testing.T) {
	threads := []StaleThread{
		{Thread: Thread{ID: "1"}, StaleReason: ""},
		{Thread: Thread{ID: "2"}, StaleReason: ""},
	}
	got := StaleOnly(threads)
	if len(got) != 0 {
		t.Errorf("StaleOnly() with none stale returned %d items, want 0", len(got))
	}
}

// ── GroupByRepoReason ─────────────────────────────────────────────────────────

func makeStaleThread(id, repo string, reason StaleReason) StaleThread {
	t := Thread{}
	t.ID = id
	t.Repository.FullName = repo

	return StaleThread{Thread: t, StaleReason: reason}
}

func TestGroupByRepoReason_Empty(t *testing.T) {
	got := GroupByRepoReason(nil)
	if len(got) != 0 {
		t.Errorf("GroupByRepoReason(nil) returned %d groups, want 0", len(got))
	}
}

func TestGroupByRepoReason_SingleGroup(t *testing.T) {
	threads := []StaleThread{
		makeStaleThread("1", "org/repo", StaleReasonPRMergedClosed),
		makeStaleThread("2", "org/repo", StaleReasonPRMergedClosed),
	}

	got := GroupByRepoReason(threads)

	if len(got) != 1 {
		t.Fatalf("expected 1 group, got %d", len(got))
	}
	if got[0].Repo != "org/repo" {
		t.Errorf("Repo = %q, want %q", got[0].Repo, "org/repo")
	}
	if got[0].StaleReason != StaleReasonPRMergedClosed {
		t.Errorf("StaleReason = %q, want %q", got[0].StaleReason, StaleReasonPRMergedClosed)
	}
	if len(got[0].Threads) != 2 {
		t.Errorf("Threads count = %d, want 2", len(got[0].Threads))
	}
}

func TestGroupByRepoReason_MultipleGroups_SortedBySize(t *testing.T) {
	threads := []StaleThread{
		// small group: 1 thread
		makeStaleThread("1", "org/small", StaleReasonIssueClosed),
		// large group: 3 threads
		makeStaleThread("2", "org/large", StaleReasonPRMergedClosed),
		makeStaleThread("3", "org/large", StaleReasonPRMergedClosed),
		makeStaleThread("4", "org/large", StaleReasonPRMergedClosed),
	}

	got := GroupByRepoReason(threads)

	if len(got) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(got))
	}
	// Largest group should come first.
	if got[0].Repo != "org/large" {
		t.Errorf("first group Repo = %q, want org/large", got[0].Repo)
	}
	if len(got[0].Threads) != 3 {
		t.Errorf("first group size = %d, want 3", len(got[0].Threads))
	}
	if got[1].Repo != "org/small" {
		t.Errorf("second group Repo = %q, want org/small", got[1].Repo)
	}
}

func TestGroupByRepoReason_SameRepoTwoReasons(t *testing.T) {
	threads := []StaleThread{
		makeStaleThread("1", "org/repo", StaleReasonPRMergedClosed),
		makeStaleThread("2", "org/repo", StaleReasonIssueClosed),
		makeStaleThread("3", "org/repo", StaleReasonIssueClosed),
	}

	got := GroupByRepoReason(threads)

	// Two distinct (repo, reason) keys → 2 groups.
	if len(got) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(got))
	}
	// The IssueClosed group has 2 threads; PRMergedClosed has 1 — sorted descending.
	if got[0].StaleReason != StaleReasonIssueClosed {
		t.Errorf("first group reason = %q, want %q", got[0].StaleReason, StaleReasonIssueClosed)
	}
	if len(got[0].Threads) != 2 {
		t.Errorf("first group size = %d, want 2", len(got[0].Threads))
	}
}

// ── checkThread (unit-level via exported helpers) ─────────────────────────────

func TestCheckThread_EmptyURL(t *testing.T) {
	// A thread with no subject URL should never be stale.
	thread := Thread{}
	thread.Subject.Type = "PullRequest"
	thread.Subject.URL = ""
	thread.Reason = "review_requested"

	// checkThread is unexported but in same package.
	got := checkThread(nil, thread, "alice")
	if got != "" {
		t.Errorf("checkThread with empty URL = %q, want empty", got)
	}
}

func TestCheckThread_UnknownSubjectType(t *testing.T) {
	thread := Thread{}
	thread.Subject.Type = "Release"
	thread.Subject.URL = "https://api.github.com/repos/o/r/releases/1"
	thread.Reason = "subscribed"

	got := checkThread(nil, thread, "alice")
	if got != "" {
		t.Errorf("checkThread with Release type = %q, want empty", got)
	}
}
