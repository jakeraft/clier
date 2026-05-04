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
// typically a Problem+JSON document — preserved for callers that want to
// inspect the full envelope (e.g. validation error tables). The default
// Error() rendering is the human-readable summary, not the raw JSON, so
// stderr stays scannable.
type Error struct {
	StatusCode int
	Body       string
}

// Error renders one human-readable line: "<status> <title>: <detail>".
// Falls back to the raw body when the payload isn't ProblemDetails so we
// never lose information. Per-field validation errors get appended in
// parentheses so the user sees what to fix without re-reading the JSON.
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
	out := fmt.Sprintf("%d %s", e.StatusCode, title)
	if p.Detail != "" {
		out += ": " + p.Detail
	}
	if len(p.Errors) > 0 {
		fields := make([]string, 0, len(p.Errors))
		for _, fe := range p.Errors {
			if fe.Field != "" {
				fields = append(fields, fmt.Sprintf("%s: %s", fe.Field, fe.Detail))
			} else {
				fields = append(fields, fe.Detail)
			}
		}
		out += " (" + strings.Join(fields, "; ") + ")"
	}
	return out
}

// problem mirrors the RFC 9457 fields the CLI consumes. `Errors` is the
// per-field validation slice the server emits for 422 responses.
type problem struct {
	Type   string         `json:"type"`
	Title  string         `json:"title"`
	Status int            `json:"status"`
	Code   string         `json:"code"`
	Detail string         `json:"detail"`
	Errors []problemField `json:"errors"`
}

type problemField struct {
	Field  string `json:"field"`
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
	if p.Title == "" && p.Detail == "" && p.Code == "" && len(p.Errors) == 0 {
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

// Raw returns the unmodified server response body. Useful when the
// caller wants to dump the full ProblemDetails envelope (e.g. behind
// a `--verbose` flag) instead of the summary line.
func (e *Error) Raw() string {
	if e == nil {
		return ""
	}
	return e.Body
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
