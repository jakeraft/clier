package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/api"
	"github.com/jakeraft/clier/internal/auth"
	"github.com/jakeraft/clier/internal/config"
	"github.com/jakeraft/clier/internal/git"
	"github.com/jakeraft/clier/internal/runner"
	"github.com/jakeraft/clier/internal/runplan"
	"github.com/jakeraft/clier/internal/tmux"
)

// emit writes a value as compact JSON to w, followed by a newline.
// All success output goes through this so agents can parse the CLI
// uniformly.
func emit(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// loadConfig returns the resolved Paths bundle (env-overridable).
func loadConfig() (*config.Paths, error) {
	return config.Default()
}

// loadCredentials returns credentials or (nil, nil) when not logged in.
func loadCredentials(path string) (*auth.Credentials, error) {
	creds, err := auth.LoadCredentials(path)
	if errors.Is(err, auth.ErrNotLoggedIn) {
		return nil, nil
	}
	return creds, err
}

// newAPIClient returns an api.Client preloaded with the persisted token
// (empty string if logged out).
func newAPIClient() (*api.Client, *config.Paths, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, nil, err
	}
	creds, err := loadCredentials(cfg.CredentialsPath)
	if err != nil {
		return nil, nil, err
	}
	token := ""
	if creds != nil {
		token = creds.Token
	}
	return api.New(cfg.ServerURL, token), cfg, nil
}

// newRunner wires the orchestrator with real adapters.
func newRunner() (*runner.Runner, error) {
	client, cfg, err := newAPIClient()
	if err != nil {
		return nil, err
	}
	store := runplan.NewStore(cfg.RunsDir)
	return runner.New(runner.Deps{
		API:   client,
		Git:   git.New(),
		Tmux:  tmux.New(),
		Store: store,
	}), nil
}

// splitTeamID parses "namespace/name" — the canonical team URL form on the
// server and dashboard. The workspace-flat slug "namespace.name" is only
// used inside the runtime layer (tmux window names, agent IDs); operators
// always type the slash form.
func splitTeamID(raw string) (namespace, name string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", errors.New("team ID is required (format: namespace/name)")
	}
	i := strings.Index(raw, "/")
	if i <= 0 || i >= len(raw)-1 {
		return "", "", fmt.Errorf("invalid team ID %q (expected namespace/name)", raw)
	}
	return raw[:i], raw[i+1:], nil
}

// readContent returns content from arg[0] or stdin (when empty/missing).
func readContent(args []string) (string, error) {
	if len(args) > 0 && args[0] != "-" {
		return args[0], nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(b))
	if content == "" {
		return "", errors.New("message content is empty")
	}
	return content, nil
}
