package config

import (
	"fmt"
	"net/url"
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
// Server / dashboard URLs are validated upfront — an `http://` scheme on
// a non-loopback host is rejected so the Bearer session token never
// crosses the wire in plaintext. Loopback (`localhost`, `127.0.0.1`,
// `[::1]`) is the explicit dev exception.
func Default() (*Paths, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	server, err := resolveURL(envServerURL, DefaultServerURL)
	if err != nil {
		return nil, fmt.Errorf("server url: %w", err)
	}
	dashboard, err := resolveURL(envDashboardURL, DefaultDashboardURL)
	if err != nil {
		return nil, fmt.Errorf("dashboard url: %w", err)
	}
	base := filepath.Join(u.HomeDir, dotDir)
	return &Paths{
		BaseDir:         base,
		CredentialsPath: filepath.Join(base, "credentials.json"),
		RunsDir:         filepath.Join(base, "runs"),
		ServerURL:       server,
		DashboardURL:    dashboard,
	}, nil
}

func resolveURL(envKey, fallback string) (string, error) {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		raw = fallback
	}
	raw = strings.TrimRight(raw, "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse %s=%q: %w", envKey, raw, err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("scheme must be https (or http on loopback): %s=%q", envKey, raw)
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return "", fmt.Errorf(
			"http:// rejected for non-loopback host (Bearer token would travel in plaintext): %s=%q",
			envKey, raw,
		)
	}
	return raw, nil
}

func isLoopbackHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}
