package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	dotDir              = ".clier"
	DefaultServerURL    = "http://localhost:8080"
	DefaultDashboardURL = "http://localhost:5173"
	envServerURL        = "CLIER_SERVER_URL"
	envDashboardURL     = "CLIER_DASHBOARD_URL"
)

// Paths bundles every filesystem path and remote URL the CLI needs.
// Server URL and dashboard URL fall back to the local-dev defaults so the
// out-of-the-box `make dev` setup works without a config file.
type Paths struct {
	BaseDir         string // ~/.clier
	CredentialsPath string // ~/.clier/credentials.json
	RunsDir         string // ~/.clier/runs
	ServerURL       string
	DashboardURL    string
}

// Default returns the canonical config rooted at the current user's home.
func Default() (*Paths, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	base := filepath.Join(u.HomeDir, dotDir)
	return &Paths{
		BaseDir:         base,
		CredentialsPath: filepath.Join(base, "credentials.json"),
		RunsDir:         filepath.Join(base, "runs"),
		ServerURL:       envOr(envServerURL, DefaultServerURL),
		DashboardURL:    envOr(envDashboardURL, DefaultDashboardURL),
	}, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return strings.TrimRight(v, "/")
	}
	return fallback
}
