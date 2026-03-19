package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Thread represents a GitHub notification thread.
type Thread struct {
	ID         string     `json:"id"`
	Unread     bool       `json:"unread"`
	Reason     string     `json:"reason"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastReadAt *time.Time `json:"last_read_at"`
	URL        string     `json:"url"`
	Subject    struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Type  string `json:"type"` // Issue, PullRequest, Release, ...
	} `json:"subject"`
	Repository struct {
		FullName string `json:"full_name"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

// PullRequest holds the fields we need when checking PR state.
type PullRequest struct {
	State  string `json:"state"` // "open" | "closed"
	Merged bool   `json:"merged"`
}

// Issue holds the fields we need when checking issue state.
type Issue struct {
	State string `json:"state"` // "open" | "closed"
}

// PRReview holds the fields we need from a pull request review.
type PRReview struct {
	State string `json:"state"` // "APPROVED", "CHANGES_REQUESTED", "COMMENTED", "PENDING", "DISMISSED"
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
}

// AuthenticatedUser holds the login of the currently authenticated user.
type AuthenticatedUser struct {
	Login string `json:"login"`
}

// ListAll fetches all notifications (read + unread) and applies the filter.
func (c *Client) ListAll(f Filter) ([]Thread, error) {
	var all []Thread

	err := c.pagedGet("notifications?all=true&per_page=50", func(page []Thread) {
		for _, n := range page {
			if f.matches(n) {
				all = append(all, n)
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	return all, nil
}

// ListUnread fetches only unread notifications and applies the filter.
func (c *Client) ListUnread(f Filter) ([]Thread, error) {
	var all []Thread

	err := c.pagedGet("notifications?all=false&per_page=50", func(page []Thread) {
		for _, n := range page {
			if f.matches(n) {
				all = append(all, n)
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("list unread notifications: %w", err)
	}

	return all, nil
}

// MarkRead marks a single thread as read via PATCH /notifications/threads/{id}.
func (c *Client) MarkRead(threadID string) error {
	resp, err := c.rest.Request(http.MethodPatch, "notifications/threads/"+threadID, bytes.NewReader([]byte("{}")))
	if err != nil {
		return fmt.Errorf("mark read %s: %w", threadID, err)
	}

	defer func() { _ = resp.Body.Close() }()

	// 205 Reset Content is the success response.
	if resp.StatusCode != http.StatusResetContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mark read %s: unexpected status %d", threadID, resp.StatusCode)
	}

	return nil
}

// Done deletes the subscription for a thread ("Done" / archive).
// DELETE /notifications/threads/{id}/subscription → 204
func (c *Client) Done(threadID string) error {
	resp, err := c.rest.Request(http.MethodDelete, "notifications/threads/"+threadID+"/subscription", nil)
	if err != nil {
		return fmt.Errorf("done %s: %w", threadID, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("done %s: unexpected status %d", threadID, resp.StatusCode)
	}

	return nil
}

// Mute sets ignored=true on the thread subscription.
// PUT /notifications/threads/{id}/subscription → 200
func (c *Client) Mute(threadID string) error {
	body, _ := json.Marshal(map[string]bool{"ignored": true})

	resp, err := c.rest.Request(http.MethodPut, "notifications/threads/"+threadID+"/subscription", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mute %s: %w", threadID, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mute %s: unexpected status %d", threadID, resp.StatusCode)
	}

	return nil
}

// GetPR fetches a pull request's state and merged flag.
// subjectURL is the API URL from Thread.Subject.URL,
// e.g. https://api.github.com/repos/owner/repo/pulls/123
func (c *Client) GetPR(subjectURL string) (PullRequest, error) {
	// go-gh REST client takes a path relative to the API root.
	// Strip the API base from the full URL.
	path := stripAPIBase(subjectURL)

	var pr PullRequest
	if err := c.rest.Get(path, &pr); err != nil {
		return PullRequest{}, fmt.Errorf("get PR %s: %w", path, err)
	}

	return pr, nil
}

// GetPRReviews fetches all reviews for a pull request.
// subjectURL is the API URL from Thread.Subject.URL (the PR URL).
func (c *Client) GetPRReviews(subjectURL string) ([]PRReview, error) {
	path := stripAPIBase(subjectURL) + "/reviews"

	var reviews []PRReview
	if err := c.rest.Get(path, &reviews); err != nil {
		return nil, fmt.Errorf("get PR reviews %s: %w", path, err)
	}

	return reviews, nil
}

// GetIssue fetches an issue's state.
// subjectURL is the API URL from Thread.Subject.URL,
// e.g. https://api.github.com/repos/owner/repo/issues/123
func (c *Client) GetIssue(subjectURL string) (Issue, error) {
	path := stripAPIBase(subjectURL)

	var issue Issue
	if err := c.rest.Get(path, &issue); err != nil {
		return Issue{}, fmt.Errorf("get issue %s: %w", path, err)
	}

	return issue, nil
}

// GetAuthenticatedUser returns the login of the currently authenticated user.
func (c *Client) GetAuthenticatedUser() (string, error) {
	var user AuthenticatedUser
	if err := c.rest.Get("user", &user); err != nil {
		return "", fmt.Errorf("get authenticated user: %w", err)
	}

	return user.Login, nil
}

// stripAPIBase removes the scheme+host prefix so the path is relative to the
// API root. Handles both github.com and GHES base URLs.
func stripAPIBase(url string) string {
	// Remove scheme
	s := url
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}

	// Remove host (up to first slash after host)
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[i+1:]
	}

	// For github.com the path is already repos/... but for GHES it may be
	// prefixed with "api/v3/" — strip that prefix if present.
	s = strings.TrimPrefix(s, "api/v3/")

	return s
}

// Stats holds aggregated notification statistics for a single repository.
type Stats struct {
	Repo       string
	Total      int
	Unread     int
	ByReason   map[string]int
	Suggestion string
}

// ComputeStats aggregates notifications into per-repo Stats and attaches
// a subscription suggestion based on patterns.
func ComputeStats(threads []Thread) []Stats {
	byRepo := make(map[string]*Stats)

	for _, t := range threads {
		repo := t.Repository.FullName
		if _, ok := byRepo[repo]; !ok {
			byRepo[repo] = &Stats{
				Repo:     repo,
				ByReason: make(map[string]int),
			}
		}

		s := byRepo[repo]
		s.Total++

		if t.Unread {
			s.Unread++
		}

		s.ByReason[t.Reason]++
	}

	result := make([]Stats, 0, len(byRepo))

	for _, s := range byRepo {
		s.Suggestion = suggest(s)
		result = append(result, *s)
	}

	// Sort descending by total.
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Total > result[j-1].Total; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	return result
}

func suggest(s *Stats) string {
	subscribed := s.ByReason["subscribed"]
	reviewReq := s.ByReason["review_requested"]
	ciActivity := s.ByReason["ci_activity"]
	total := s.Total

	switch {
	case total >= 20 && subscribed > total/2:
		return "Consider unwatching this repo (Settings → Unwatch)"
	case total >= 10 && reviewReq > total/2:
		return "High review traffic — mute threads after reviewing"
	case ciActivity > total/2:
		return "Mostly CI — disable CI notifications in watch settings"
	case total >= 10 && s.Unread == 0:
		return "All read — run 'done' to archive and unsubscribe"
	default:
		return ""
	}
}
