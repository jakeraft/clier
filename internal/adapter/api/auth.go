package api

import "time"

// DeviceCodeResponse is the server's device flow initiation response.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DevicePollResponse is the server's device flow poll response.
type DevicePollResponse struct {
	Status      string        `json:"status,omitempty"`
	AccessToken string        `json:"access_token,omitempty"`
	User        *UserResponse `json:"user,omitempty"`
}

// UserResponse is the server's JSON representation of the current user.
type UserResponse struct {
	ID         int64     `json:"id"`
	Login      string    `json:"login"`
	Email      *string   `json:"email,omitempty"`
	AvatarURL  *string   `json:"avatar_url,omitempty"`
	GitHubID   *int64    `json:"github_id,omitempty"`
	Type       int       `json:"type"`
	Visibility int       `json:"visibility"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RequestDeviceCode starts the GitHub Device Flow.
// POST /auth/device
func (c *Client) RequestDeviceCode() (*DeviceCodeResponse, error) {
	var r DeviceCodeResponse
	return &r, c.post("/auth/device", nil, &r)
}

// PollDeviceAuth polls for device flow completion.
// POST /auth/device/poll
func (c *Client) PollDeviceAuth(deviceCode string) (*DevicePollResponse, error) {
	var r DevicePollResponse
	return &r, c.post("/auth/device/poll", map[string]string{"device_code": deviceCode}, &r)
}

// GetCurrentUser returns the authenticated user.
// GET /api/v1/user
func (c *Client) GetCurrentUser() (*UserResponse, error) {
	var r UserResponse
	return &r, c.get("/api/v1/user", &r)
}
