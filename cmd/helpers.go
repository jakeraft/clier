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
	"github.com/spf13/cobra"
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

// requireOneArg is a cobra.Args validator that takes the place of
// cobra.ExactArgs(1) so the missing/extra argument message matches the
// rest of the CLI's tone. cobra.ExactArgs emits "accepts 1 arg(s),
// received 0" which reads like an internal API trace; the human-friendly
// label name lets us write "<run-id> is required" or similar.
func requireOneArg(label string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch len(args) {
		case 0:
			return fmt.Errorf("%s is required\n\nUsage:\n  %s", label, cmd.UseLine())
		case 1:
			return nil
		default:
			return fmt.Errorf("expected exactly one %s, got %d\n\nUsage:\n  %s", label, len(args), cmd.UseLine())
		}
	}
}

// readContent returns the message content from arg[0] or, when arg[0] is
// missing or "-", from stdin. Both paths apply the same emptiness check
// so callers like `clier run tell --run X --to Y ""` fail with a precise
// "message content is empty" before any downstream lookup (the previous
// implementation only trimmed the stdin path, so an empty arg fell
// through to runner.Tell and surfaced as a misleading "run not found").
func readContent(args []string) (string, error) {
	if len(args) > 0 && args[0] != "-" {
		content := strings.TrimSpace(args[0])
		if content == "" {
			return "", errors.New("message content is empty")
		}
		return content, nil
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
