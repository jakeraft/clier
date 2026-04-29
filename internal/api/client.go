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

// Error wraps a non-2xx response. Body is the raw server payload — typically
// a Problem+JSON document. The CLI surfaces it verbatim instead of
// translating reasons locally (server is the SSOT for error wording).
type Error struct {
	StatusCode int
	Body       string
}

func (e *Error) Error() string {
	return fmt.Sprintf("server returned %d: %s", e.StatusCode, e.Body)
}

// problem is the subset of RFC 7807 the CLI needs for branching (e.g. retry
// on FAILED_PRECONDITION during device-flow polling). Other fields are
// preserved in Body for display.
type problem struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Code returns the structured Problem code if the body parses as one.
func (e *Error) Code() string {
	if e == nil {
		return ""
	}
	var p problem
	if err := json.Unmarshal([]byte(e.Body), &p); err != nil {
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
