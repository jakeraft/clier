package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	dotDir          = ".clier"
	envServerURL    = "CLIER_SERVER_URL"
	envDashboardURL = "CLIER_DASHBOARD_URL"
)

// DefaultServerURL / DefaultDashboardURL are var (not const) so the build
// pipeline can override them via -ldflags="-X .../config.DefaultServerURL=…".
// Default is the prod surface so a plain `go install` / brew install /
// source build all point at production. The dev binary (`clier-dev` from
// `make install-dev`) overrides both to localhost.
var (
	DefaultServerURL    = "https://www.clier.jakeraft.com"
	DefaultDashboardURL = "https://www.clier.jakeraft.com"
)

// Paths bundles every filesystem path and remote URL the CLI needs.
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
