package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/domain"
)

// Credentials stores the user's authentication state.
type Credentials struct {
	Token string `json:"token"`
	Login string `json:"login"`
}

// Load reads credentials from the given path.
func Load(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &domain.Fault{Kind: domain.KindAuthRequired, Cause: err}
		}
		return nil, fmt.Errorf("read credentials: %w", err)
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
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
