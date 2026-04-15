package api

import "time"

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type DevicePollResponse struct {
	Status      string        `json:"status"`
	AccessToken string        `json:"access_token"`
	User        *UserResponse `json:"user"`
}

type UserResponse struct {
	Name       string    `json:"name"`
	Email      string    `json:"email,omitempty"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	GitHubID   *int64    `json:"github_id,omitempty"`
	Type       int       `json:"type"`
	Visibility int       `json:"visibility"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (c *Client) RequestDeviceCode() (*DeviceCodeResponse, error) {
	var r DeviceCodeResponse
	return &r, c.post("/auth/device", nil, &r)
}

func (c *Client) PollDeviceAuth(deviceCode string) (*DevicePollResponse, error) {
	var r DevicePollResponse
	body := map[string]string{"device_code": deviceCode}
	return &r, c.post("/auth/device/poll", body, &r)
}

func (c *Client) GetCurrentUser() (*UserResponse, error) {
	var r UserResponse
	return &r, c.get("/api/v1/user", &r)
}

func (c *Client) Logout() error {
	return c.post("/api/v1/auth/logout", nil, nil)
}
