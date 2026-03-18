package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps the go-gh REST client with helper methods.
type Client struct {
	rest *ghapi.RESTClient
	host string
}

// NewClient constructs a Client for the given host. An empty host uses the
// default host from the user's gh config.
func NewClient(host string) (*Client, error) {
	opts := ghapi.ClientOptions{}
	if host != "" {
		opts.Host = host
	}

	rest, err := ghapi.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("create REST client: %w", err)
	}

	return &Client{rest: rest, host: host}, nil
}

// RESTClient exposes the underlying client for callers that need raw access.
func (c *Client) RESTClient() *ghapi.RESTClient {
	return c.rest
}

// pagedGet fetches all pages of a GET endpoint that uses Link-header
// pagination and accumulates results into dest (which must be a pointer to a
// slice). It calls appendFn on each page's raw JSON slice so callers control
// the target type.
func (c *Client) pagedGet(path string, appendFn func(page []Thread)) error {
	url := path
	for url != "" {
		var page []Thread
		resp, err := c.rest.Request(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("GET %s: %w", url, err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read body %s: %w", url, err)
		}

		if err := json.Unmarshal(body, &page); err != nil {
			return fmt.Errorf("decode %s: %w", url, err)
		}

		appendFn(page)

		url = nextLink(resp.Header.Get("Link"))
	}

	return nil
}

// nextLink parses the RFC 5988 Link header and returns the URL for rel="next",
// or an empty string if there is no next page.
func nextLink(header string) string {
	if header == "" {
		return ""
	}

	// Simple parser: split on ", " between entries, look for rel="next".
	for _, part := range splitLinkHeader(header) {
		url, rel := parseLinkEntry(part)
		if rel == "next" {
			return url
		}
	}

	return ""
}

func splitLinkHeader(h string) []string {
	var parts []string
	depth := 0
	start := 0

	for i := 0; i < len(h); i++ {
		switch h[i] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, h[start:i])
				start = i + 1
			}
		}
	}

	parts = append(parts, h[start:])

	return parts
}

func parseLinkEntry(entry string) (url, rel string) {
	// entry looks like: <https://api.github.com/...>; rel="next"
	gtIdx := -1

	for i, c := range entry {
		if c == '>' {
			gtIdx = i
			break
		}
	}

	if gtIdx < 0 {
		return "", ""
	}

	ltIdx := -1

	for i := 0; i < gtIdx; i++ {
		if entry[i] == '<' {
			ltIdx = i
			break
		}
	}

	if ltIdx < 0 {
		return "", ""
	}

	url = entry[ltIdx+1 : gtIdx]
	rest := entry[gtIdx+1:]

	// Find rel="..."
	const relKey = `rel="`
	idx := 0

	for i := 0; i+len(relKey) <= len(rest); i++ {
		if rest[i:i+len(relKey)] == relKey {
			idx = i + len(relKey)
			end := idx

			for end < len(rest) && rest[end] != '"' {
				end++
			}

			rel = rest[idx:end]

			break
		}
	}

	return url, rel
}

// Filter holds optional filtering criteria for listing notifications.
type Filter struct {
	Repo      string        // "owner/repo" — empty means all repos
	Org       string        // org login — empty means all orgs
	OlderThan time.Duration // only include notifications older than this
	OnlyRead  bool          // only include already-read notifications
}

// matches reports whether n passes the filter.
func (f Filter) matches(n Thread) bool {
	if f.Repo != "" && n.Repository.FullName != f.Repo {
		return false
	}

	if f.Org != "" {
		owner := n.Repository.Owner.Login
		if owner != f.Org {
			return false
		}
	}

	if f.OlderThan > 0 {
		cutoff := time.Now().Add(-f.OlderThan)
		if !n.UpdatedAt.Before(cutoff) {
			return false
		}
	}

	if f.OnlyRead && n.Unread {
		return false
	}

	return true
}
