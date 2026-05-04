package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// requestTimeout caps every API call so a hung server can't freeze the CLI.
// Long enough for cold cold-cache resolves; short enough that operators
// notice and Ctrl-C themselves rather than waiting forever.
const requestTimeout = 30 * time.Second

// Client is a thin HTTP client for clier-server. Authentication is a single
// bearer token presented on every request; an empty token sends no
// Authorization header (public endpoints).
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

// Error wraps a non-2xx response. Body is the raw server payload —
// typically a Problem+JSON document — preserved as a public field for
// callers that want the full envelope (e.g. behind a `--verbose` flag).
// Error() renders the human-readable summary; Code() exposes the
// structured taxonomy slug for branching.
type Error struct {
	StatusCode int
	Body       string
}

// Error renders one human-readable line: "<status> <title>: <detail>".
// The server is the SSOT for both fields — title is the short category
// label, detail is the per-occurrence one-liner that already covers
// per-field violations (the server composes detail from errors[] so the
// CLI does not have to). Falls back to the raw body when the payload
// isn't ProblemDetails so no information is lost.
func (e *Error) Error() string {
	p, ok := e.problem()
	if !ok {
		return fmt.Sprintf("server returned %d: %s", e.StatusCode, e.Body)
	}
	title := p.Title
	if title == "" {
		title = http.StatusText(e.StatusCode)
		if title == "" {
			title = fmt.Sprintf("HTTP %d", e.StatusCode)
		}
	}
	if p.Detail == "" {
		return fmt.Sprintf("%d %s", e.StatusCode, title)
	}
	return fmt.Sprintf("%d %s: %s", e.StatusCode, title, p.Detail)
}

// problem captures the three RFC 9457 fields the CLI actually consumes.
// Type/instance are preserved on Body for verbose inspection but ignored
// at the Error() / Code() entry points.
type problem struct {
	Title  string `json:"title"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *Error) problem() (problem, bool) {
	var p problem
	if err := json.Unmarshal([]byte(e.Body), &p); err != nil {
		return problem{}, false
	}
	// A non-ProblemDetails JSON body (e.g. {"foo":1}) parses without
	// error but yields zero-value fields. Treat as not-a-problem so the
	// caller falls back to raw body.
	if p.Title == "" && p.Detail == "" && p.Code == "" {
		return problem{}, false
	}
	return p, true
}

// Code returns the structured Problem code if the body parses as one.
// Used for branching on transient conditions (e.g. FAILED_PRECONDITION
// during device-flow polling).
func (e *Error) Code() string {
	if e == nil {
		return ""
	}
	p, ok := e.problem()
	if !ok {
		return ""
	}
	return p.Code
}

func (c *Client) do(method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return &Error{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
