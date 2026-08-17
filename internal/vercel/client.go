// Package vercel is the platform-integration adapter for the Vercel API.
// It knows how to talk HTTP to Vercel and shape the response into plain
// data (map[string]string plus a list of redacted keys) — it contains no
// diffing or validation logic. That all lives in internal/core, which has
// never heard of Vercel.
package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.vercel.com"

// Client talks to the Vercel REST API to fetch a project's configured
// environment variables.
type Client struct {
	BaseURL string
	Token   string
	TeamID  string
	HTTP    *http.Client
}

// NewClient builds a Client authenticated with the given API token. token
// must be non-empty; Vercel's API rejects unauthenticated requests, and
// failing fast here gives a clearer error than an eventual 403.
func NewClient(token string) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("vercel: API token is empty")
	}
	return &Client{
		BaseURL: defaultBaseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}, nil
}

type envVarsResponse struct {
	Envs []envVar `json:"envs"`
}

type envVar struct {
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Type      string   `json:"type"`
	Target    []string `json:"target"`
	Sensitive bool     `json:"sensitive"`
}

// FetchEnv retrieves the environment variables configured for projectIDOrName
// that apply to the given target (e.g. "production", "preview",
// "development"), as they'd actually be injected into that deployment.
//
// It returns two things: values holds every variable whose value Vercel
// was willing to return, and redacted lists keys that exist on the
// project for this target but whose value the API withheld (Vercel does
// this for variables marked "sensitive") — Relay can still report that
// they're present, just not validate their contents.
func (c *Client) FetchEnv(ctx context.Context, projectIDOrName, target string) (values map[string]string, redacted []string, err error) {
	if strings.TrimSpace(projectIDOrName) == "" {
		return nil, nil, fmt.Errorf("vercel: project ID or name is empty")
	}
	if strings.TrimSpace(target) == "" {
		return nil, nil, fmt.Errorf("vercel: target is empty")
	}

	reqURL, err := url.Parse(fmt.Sprintf("%s/v9/projects/%s/env", c.BaseURL, url.PathEscape(projectIDOrName)))
	if err != nil {
		return nil, nil, fmt.Errorf("vercel: building request URL: %w", err)
	}
	q := reqURL.Query()
	q.Set("decrypt", "true")
	if c.TeamID != "" {
		q.Set("teamId", c.TeamID)
	}
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("vercel: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("vercel: request to %s failed: %w", reqURL.String(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("vercel: reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("vercel: %s %s returned %d: %s", req.Method, reqURL.Path, resp.StatusCode, describeAPIError(body))
	}

	var parsed envVarsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, nil, fmt.Errorf("vercel: unexpected response shape from %s: %w", reqURL.Path, err)
	}

	values = make(map[string]string)
	for _, ev := range parsed.Envs {
		if !targetsInclude(ev.Target, target) {
			continue
		}
		switch {
		case ev.Value != "":
			values[ev.Key] = ev.Value
		case ev.Type != "plain" || ev.Sensitive:
			// Value withheld by the API (sensitive/encrypted var); we know
			// it exists but can't inspect it.
			redacted = append(redacted, ev.Key)
		default:
			// A genuinely empty "plain" value — real, and worth flagging
			// by the core validator, not hidden as if it were redacted.
			values[ev.Key] = ""
		}
	}

	return values, redacted, nil
}

func targetsInclude(targets []string, want string) bool {
	for _, t := range targets {
		if strings.EqualFold(t, want) {
			return true
		}
	}
	return false
}

// describeAPIError best-effort extracts Vercel's {"error":{"message":...}}
// shape, falling back to the raw body when it doesn't parse.
func describeAPIError(body []byte) string {
	var withMessage struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &withMessage); err == nil && withMessage.Error.Message != "" {
		return withMessage.Error.Message
	}
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) > 300 {
		trimmed = trimmed[:300] + "..."
	}
	return trimmed
}
