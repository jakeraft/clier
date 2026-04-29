package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jakeraft/clier/internal/api"
)

// Credentials is the persisted bearer token + the namespace that minted it.
type Credentials struct {
	Token string `json:"token"`
	Login string `json:"login"`
}

// ErrNotLoggedIn is returned when the credentials file is absent.
var ErrNotLoggedIn = errors.New("not logged in")

// LoadCredentials reads ~/.clier/credentials.json. Returns ErrNotLoggedIn
// when the file does not exist.
func LoadCredentials(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotLoggedIn
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("corrupted credentials file %s: %w", path, err)
	}
	return &c, nil
}

// SaveCredentials writes credentials with 0600 permission, creating parent
// dirs as needed.
func SaveCredentials(path string, c *Credentials) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// DeleteCredentials removes the credentials file. Missing file is not an
// error — logout is idempotent.
func DeleteCredentials(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}

// LoginPrompt is the message Login surfaces to the user during device flow.
type LoginPrompt struct {
	UserCode        string
	VerificationURI string
}

// Login drives the GitHub device flow against the API client and persists
// the resulting session token. The notify callback is invoked once with the
// user_code + verification_uri the operator must visit.
func Login(client *api.Client, credentialsPath string, notify func(LoginPrompt)) (*api.Namespace, error) {
	auth, err := client.AuthDeviceStart()
	if err != nil {
		return nil, fmt.Errorf("start device flow: %w", err)
	}
	if notify != nil {
		notify(LoginPrompt{UserCode: auth.UserCode, VerificationURI: auth.VerificationURI})
	}

	interval := time.Duration(auth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(auth.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		sess, err := client.AuthDeviceComplete(auth.DeviceCode)
		if err == nil {
			creds := &Credentials{Token: sess.SessionToken, Login: sess.Namespace.Name}
			if err := SaveCredentials(credentialsPath, creds); err != nil {
				return nil, fmt.Errorf("save credentials: %w", err)
			}
			ns := sess.Namespace
			return &ns, nil
		}

		var apiErr *api.Error
		if errors.As(err, &apiErr) && apiErr.Code() == "FAILED_PRECONDITION" {
			// User has not yet confirmed. Keep polling.
			continue
		}
		return nil, fmt.Errorf("complete device flow: %w", err)
	}

	return nil, errors.New("authentication timed out — re-run `clier auth login`")
}
