package tui

import (
	"testing"
	"time"
)

// ── humanAge ─────────────────────────────────────────────────────────────────

func TestHumanAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "just now (30s ago)",
			t:    now.Add(-30 * time.Second),
			want: "just now",
		},
		{
			name: "minutes ago",
			t:    now.Add(-5 * time.Minute),
			want: "5m ago",
		},
		{
			name: "hours ago",
			t:    now.Add(-3 * time.Hour),
			want: "3h ago",
		},
		{
			name: "days ago",
			t:    now.Add(-10 * 24 * time.Hour),
			want: "10d ago",
		},
		{
			name: "months ago",
			t:    now.Add(-60 * 24 * time.Hour),
			want: "2mo ago",
		},
		{
			name: "years ago",
			t:    now.Add(-400 * 24 * time.Hour),
			want: "1y ago",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := humanAge(tc.t)
			if got != tc.want {
				t.Errorf("humanAge() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ── truncate ─────────────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{
			name:  "shorter than max — unchanged",
			input: "hello",
			max:   10,
			want:  "hello",
		},
		{
			name:  "exactly max — unchanged",
			input: "hello",
			max:   5,
			want:  "hello",
		},
		{
			name:  "longer than max — truncated with ellipsis",
			input: "hello world",
			max:   8,
			want:  "hello w…",
		},
		{
			name:  "max of 1 — just ellipsis",
			input: "abcdef",
			max:   1,
			want:  "…",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.input, tc.max)
			if got != tc.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
			}
		})
	}
}

// ── topKey ────────────────────────────────────────────────────────────────────

func TestTopKey(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]int
		want  string
	}{
		{
			name:  "empty map returns empty string",
			input: map[string]int{},
			want:  "",
		},
		{
			name:  "single entry",
			input: map[string]int{"subscribed": 5},
			want:  "subscribed",
		},
		{
			name:  "highest count wins",
			input: map[string]int{"subscribed": 3, "mention": 7, "review_requested": 2},
			want:  "mention",
		},
		{
			name:  "two equal counts — deterministic winner (whichever is higher)",
			input: map[string]int{"a": 5, "b": 5},
			// Both are valid; just verify the returned key is one of them.
			want: "", // handled separately below
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := topKey(tc.input)

			if tc.name == "two equal counts — deterministic winner (whichever is higher)" {
				if got != "a" && got != "b" {
					t.Errorf("topKey() = %q, want 'a' or 'b'", got)
				}
				return
			}

			if got != tc.want {
				t.Errorf("topKey() = %q, want %q", got, tc.want)
			}
		})
	}
}
