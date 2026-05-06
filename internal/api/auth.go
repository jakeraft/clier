package api

// DeviceAuthorization is the start response of the GitHub device flow.
type DeviceAuthorization struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// SessionResponse is the successful complete response: a raw bearer token
// and the namespace bound to that session.
type SessionResponse struct {
	SessionToken string    `json:"session_token"`
	Namespace    Namespace `json:"namespace"`
}

// Namespace mirrors the server's Namespace schema.
type Namespace struct {
	Name      string `json:"name"`
	GitHubID  int64  `json:"github_id"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// AuthDeviceStart kicks off the device flow.
func (c *Client) AuthDeviceStart() (*DeviceAuthorization, error) {
	var r DeviceAuthorization
	return &r, c.do("POST", "/api/v1/auth/device/start", nil, &r)
}

// AuthDeviceComplete polls once for token issuance. The server returns 412
// (FAILED_PRECONDITION) while the user has not yet confirmed.
func (c *Client) AuthDeviceComplete(deviceCode string) (*SessionResponse, error) {
	var r SessionResponse
	body := map[string]string{"device_code": deviceCode}
	return &r, c.do("POST", "/api/v1/auth/device/complete", body, &r)
}

// AuthMe returns the namespace bound to the current session.
func (c *Client) AuthMe() (*Namespace, error) {
	var r Namespace
	return &r, c.do("GET", "/api/v1/auth/me", nil, &r)
}

// AuthLogout revokes the current session.
func (c *Client) AuthLogout() error {
	return c.do("POST", "/api/v1/auth/logout", nil, nil)
}
