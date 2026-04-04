package cmd

import (
	"errors"
	"os"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/adapter/workspace"
	"github.com/jakeraft/clier/internal/app/session"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSessionCmd())
}

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
	}
	cmd.AddCommand(newSessionStartCmd())
	cmd.AddCommand(newSessionStopCmd())
	cmd.AddCommand(newSessionListCmd())
	cmd.AddCommand(newSessionSendCmd())
	cmd.AddCommand(newSessionLogCmd())
	cmd.AddCommand(newSessionLogsCmd())
	return cmd
}

func newSessionStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "start <team-id>",
		Short:       "Start a session",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			t, err := store.GetTeam(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			term := terminal.NewCmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := session.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			s, err := svc.Start(cmd.Context(), t, cfg.Auth)
			if err != nil {
				return err
			}
			return printJSON(s)
		},
	}
	return cmd
}

func newSessionStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "stop <id>",
		Short:       "Stop a session",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewCmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := session.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			if err := svc.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"stopped": args[0]})
		},
	}
}

func newSessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			sessions, err := store.ListSessions(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(sessions)
		},
	}
}

func newSessionSendCmd() *cobra.Command {
	var sessionFlag, toMemberID string

	cmd := &cobra.Command{
		Use:         "send <content>",
		Short:       "Send a message to a teammate",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID, fromMemberID, err := resolveSessionContext(sessionFlag)
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewCmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := session.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			if err := svc.Send(cmd.Context(), sessionID, fromMemberID, toMemberID, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "delivered",
				"from":   fromMemberID,
				"to":     toMemberID,
			})
		},
	}
	cmd.Flags().StringVar(&sessionFlag, "session", "", "Session ID (defaults to CLIER_SESSION_ID)")
	cmd.Flags().StringVar(&toMemberID, "to", "", "Recipient member ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newSessionLogCmd() *cobra.Command {
	var sessionFlag string

	cmd := &cobra.Command{
		Use:         "log <content>",
		Short:       "Record a session log entry",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID, memberID, err := resolveSessionContext(sessionFlag)
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewCmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := session.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			if err := svc.Log(cmd.Context(), sessionID, memberID, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status":  "logged",
				"member":  memberID,
				"session": sessionID,
			})
		},
	}
	cmd.Flags().StringVar(&sessionFlag, "session", "", "Session ID (defaults to CLIER_SESSION_ID)")
	return cmd
}

func newSessionLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <session-id>",
		Short: "List session logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			logs, err := store.ListLogsBySessionID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printJSON(logs)
		},
	}
}

// resolveSessionContext resolves session ID and member ID from env vars set by clier.
// CLIER_SESSION_ID identifies the session, CLIER_MEMBER_ID identifies the sender.
func resolveSessionContext(sessionFlag string) (sessionID, memberID string, err error) {
	sessionID = sessionFlag
	if sessionID == "" {
		sessionID = os.Getenv("CLIER_SESSION_ID")
	}
	if sessionID == "" {
		return "", "", errors.New("--session flag or CLIER_SESSION_ID must be set")
	}
	memberID = os.Getenv("CLIER_MEMBER_ID")
	return sessionID, memberID, nil
}
