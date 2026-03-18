package github

import (
	"sync"
)

// StaleReason is a human-readable explanation of why a notification is stale.
type StaleReason string

const (
	StaleReasonPRMergedClosed StaleReason = "PR merged/closed"
	StaleReasonPRApproved     StaleReason = "PR already approved by others"
	StaleReasonIssueClosed    StaleReason = "Issue closed"
)

// StaleThread pairs a notification thread with the reason it is stale.
// StaleReason is empty when the thread is not stale.
type StaleThread struct {
	Thread      Thread
	StaleReason StaleReason
}

const staleConcurrency = 10

// CheckStalePR inspects PR- and issue-related threads to determine whether
// they still require action from the authenticated user. It issues concurrent
// API calls (up to staleConcurrency at a time) and returns one StaleThread per
// input thread; entries with an empty StaleReason are not stale.
//
// viewerLogin is the authenticated user's login, used to exclude their own
// reviews when checking whether a PR is already approved by someone else.
func CheckStalePR(client *Client, threads []Thread, viewerLogin string) []StaleThread {
	results := make([]StaleThread, len(threads))

	sem := make(chan struct{}, staleConcurrency)
	var wg sync.WaitGroup

	for i, t := range threads {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, thread Thread) {
			defer wg.Done()
			defer func() { <-sem }()

			reason := checkThread(client, thread, viewerLogin)
			results[idx] = StaleThread{Thread: thread, StaleReason: reason}
		}(i, t)
	}

	wg.Wait()

	return results
}

// checkThread returns the StaleReason for a single thread, or "" if not stale.
func checkThread(client *Client, t Thread, viewerLogin string) StaleReason {
	subjectType := t.Subject.Type
	reason := t.Reason
	url := t.Subject.URL

	if url == "" {
		return ""
	}

	switch subjectType {
	case "PullRequest":
		return checkPRThread(client, reason, url, viewerLogin)
	case "Issue":
		return checkIssueThread(client, reason, url)
	}

	return ""
}

// checkPRThread handles PR-type subjects.
func checkPRThread(client *Client, reason, url, viewerLogin string) StaleReason {
	switch reason {
	case "review_requested", "assign", "mention", "subscribed", "team_mention":
		pr, err := client.GetPR(url)
		if err != nil {
			return ""
		}

		if pr.State == "closed" || pr.Merged {
			return StaleReasonPRMergedClosed
		}

		// For review_requested on an open PR: check if already approved by others.
		if reason == "review_requested" {
			reviews, err := client.GetPRReviews(url)
			if err != nil {
				return ""
			}

			if hasApprovalFromOther(reviews, viewerLogin) {
				return StaleReasonPRApproved
			}
		}
	}

	return ""
}

// checkIssueThread handles Issue-type subjects.
func checkIssueThread(client *Client, reason, url string) StaleReason {
	switch reason {
	case "assign", "mention", "subscribed", "team_mention":
		issue, err := client.GetIssue(url)
		if err != nil {
			return ""
		}

		if issue.State == "closed" {
			return StaleReasonIssueClosed
		}
	}

	return ""
}

// hasApprovalFromOther returns true if reviews contains at least one APPROVED
// review from a user other than viewerLogin. Only the latest review state per
// user is considered (later entries in the slice override earlier ones for the
// same user, matching GitHub's own rules).
func hasApprovalFromOther(reviews []PRReview, viewerLogin string) bool {
	latest := make(map[string]string) // login → latest state

	for _, r := range reviews {
		login := r.User.Login
		if login == "" {
			continue
		}

		state := r.State
		// Only track submitted (non-pending) reviews.
		if state == "PENDING" {
			continue
		}

		latest[login] = state
	}

	for login, state := range latest {
		if login != viewerLogin && state == "APPROVED" {
			return true
		}
	}

	return false
}

// StaleOnly filters a slice of StaleThread to those that are actually stale.
func StaleOnly(threads []StaleThread) []StaleThread {
	var out []StaleThread

	for _, st := range threads {
		if st.StaleReason != "" {
			out = append(out, st)
		}
	}

	return out
}

// GroupByRepoReason groups stale threads by (repository, stale reason) and
// returns the groups sorted descending by size.
func GroupByRepoReason(threads []StaleThread) []StaleGroup {
	type key struct {
		repo   string
		reason StaleReason
	}

	order := []key{}
	groups := map[key]*StaleGroup{}

	for _, st := range threads {
		k := key{repo: st.Thread.Repository.FullName, reason: st.StaleReason}
		if _, ok := groups[k]; !ok {
			groups[k] = &StaleGroup{
				Repo:        k.repo,
				StaleReason: k.reason,
			}
			order = append(order, k)
		}

		groups[k].Threads = append(groups[k].Threads, st.Thread)
	}

	result := make([]StaleGroup, 0, len(order))
	for _, k := range order {
		result = append(result, *groups[k])
	}

	// Sort descending by count.
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && len(result[j].Threads) > len(result[j-1].Threads); j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	return result
}

// StaleGroup is a collection of stale threads that share a repo and reason.
type StaleGroup struct {
	Repo        string
	StaleReason StaleReason
	Threads     []Thread
}
