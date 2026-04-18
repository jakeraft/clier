package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Error is a structured API error with the original response status.
// When the server returns a recognizable ErrorResponse envelope, Status
// is populated and command-layer code should branch on Status.Reason
// for user-friendly presentation.
type Error struct {
	StatusCode int
	Body       string
	Status     *Status
}

// Client is the HTTP client for the clier-server API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *Client) do(method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read error response: %w", err)
		}
		return &Error{StatusCode: resp.StatusCode, Body: string(b), Status: parseStatus(b)}
	}
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}
	return nil
}

func (c *Client) get(path string, result any) error         { return c.do("GET", path, nil, result) }
func (c *Client) post(path string, body, result any) error  { return c.do("POST", path, body, result) }
func (c *Client) put(path string, body, result any) error   { return c.do("PUT", path, body, result) }
func (c *Client) delete(path string) error                  { return c.do("DELETE", path, nil, nil) }
func (c *Client) patch(path string, body, result any) error { return c.do("PATCH", path, body, result) }
