package cmd

import (
	"fmt"
	"os"

	"github.com/jakeraft/clier/internal/api"
	"github.com/jakeraft/clier/internal/auth"
	"github.com/jakeraft/clier/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAuthCmd())
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Log in and manage credentials",
		Args:  cobra.ArbitraryArgs,
		RunE:  helpOrUnknown,
	}
	cmd.AddCommand(newAuthLoginCmd(), newAuthLogoutCmd(), newAuthStatusCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in via GitHub device flow",
		Args:  cobra.NoArgs,
		Long: `Log in via the GitHub device flow.

Login is only required to author teams (team create / update / delete)
or to star (team star / unstar). Browsing the catalog (team list /
team get) and starting a public run (run start) work anonymously.

If a valid session already exists this command is a no-op — it prints
the current login as JSON and exits 0 without starting a new device
flow. Use 'clier auth logout' first to switch accounts.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			// Already-logged-in fast path: validate the persisted session
			// via auth.ProbeSession. If the server agrees the session is
			// live, no-op with a stderr note + login on stdout. Probe
			// failures (UNAUTHENTICATED 또는 transport) fall through to
			// a fresh device flow — Login replaces the stale credential
			// on success.
			if creds, err := loadCredentials(cfg.CredentialsPath); err == nil && creds != nil {
				probe, perr := auth.ProbeSession(api.New(cfg.ServerURL, creds.Token), creds.Login)
				if perr == nil && probe.LoggedIn {
					fmt.Fprintf(os.Stderr, "note: already logged in as %s\n", probe.Login)
					return emit(cmd.OutOrStdout(), map[string]any{"login": probe.Login})
				}
			}
			// Public endpoint — no token needed for device flow.
			client := api.New(cfg.ServerURL, "")
			ns, err := auth.Login(client, cfg.CredentialsPath, func(prompt auth.LoginPrompt) {
				fmt.Fprintf(os.Stderr, "First, copy your one-time code: %s\n", prompt.UserCode)
				fmt.Fprintf(os.Stderr, "Then open: %s\n", prompt.VerificationURI)
				fmt.Fprintln(os.Stderr, "Waiting for confirmation...")
			})
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"login": ns.Name,
			})
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke the current session",
		Args:  cobra.NoArgs,
		Long: `Revoke the persisted session.

Best-effort server revoke + local credential delete. The local
credential is always deleted, even if the server-side revoke fails —
the user is fully logged out either way.

Idempotent on a healthy environment: safe to run when no session
exists. Note that a malformed CLIER_SERVER_URL prevents logout
because the loader still wants a valid config — fix the env first.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, cfg, err := newAPIClient()
			if err != nil {
				return err
			}
			creds, err := loadCredentials(cfg.CredentialsPath)
			if err != nil {
				return err
			}
			if creds == nil {
				return emit(cmd.OutOrStdout(), map[string]any{"logged_out": true})
			}
			// Best-effort server revoke — even if the network is down we still
			// remove the local token so the user is fully logged out.
			if err := client.AuthLogout(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: server logout failed: %s\n", err)
			}
			if err := auth.DeleteCredentials(cfg.CredentialsPath); err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{"logged_out": true})
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show login status",
		Args:  cobra.NoArgs,
		Long: `Show whether a session is active and which login it
belongs to. The server is consulted — an expired token surfaces
distinctly from a clean logged-out state, so a script can tell
"never logged in" apart from "session aged out".`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			creds, err := loadCredentials(cfg.CredentialsPath)
			if err != nil {
				return err
			}
			if creds == nil {
				return emit(cmd.OutOrStdout(), authStatus(cfg, false, "", ""))
			}
			probe, err := auth.ProbeSession(api.New(cfg.ServerURL, creds.Token), creds.Login)
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), authStatus(cfg, probe.LoggedIn, probe.Login, probe.Reason))
		},
	}
}

func authStatus(cfg *config.Paths, loggedIn bool, login, reason string) map[string]any {
	out := map[string]any{
		"logged_in":  loggedIn,
		"server_url": cfg.ServerURL,
	}
	if login != "" {
		out["login"] = login
	}
	if reason != "" {
		out["reason"] = reason
	}
	return out
}
