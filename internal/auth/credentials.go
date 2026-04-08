package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Credentials stores the user's authentication state.
type Credentials struct {
	Token string `json:"token"`
	Login string `json:"login"`
}

// DefaultPath returns ~/.clier/credentials.json.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clier", "credentials.json")
}

// Load reads credentials from the given path.
func Load(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("not logged in. Run 'clier auth login' first.")
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("corrupted credentials file: %w", err)
	}
	return &c, nil
}

// Save writes credentials to the given path with 0600 permission.
func Save(path string, creds *Credentials) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Delete removes the credentials file. No error if it doesn't exist.
func Delete(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
