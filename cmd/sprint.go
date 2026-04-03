package cmd

import (
	"errors"
	"os"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/adapter/workspace"
	"github.com/jakeraft/clier/internal/app/sprint"
	"github.com/jakeraft/clier/internal/app/team"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSprintCmd())
}

func newSprintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Manage sprints",
	}
	cmd.AddCommand(newSprintStartCmd())
	cmd.AddCommand(newSprintStopCmd())
	cmd.AddCommand(newSprintListCmd())
	cmd.AddCommand(newSprintWhoamiCmd())
	return cmd
}

func newSprintStartCmd() *cobra.Command {
	var teamID string

	cmd := &cobra.Command{
		Use:         "start",
		Short:       "Start a sprint",
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

			teamSvc := team.New(store)
			term := terminal.NewCmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := sprint.New(teamSvc, store, term, ws, cfg.Auth, cfg.Paths.Workspaces(), cfg.Paths.Home())

			sp, err := svc.Start(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			return printJSON(sp)
		},
	}
	cmd.Flags().StringVar(&teamID, "team", "", "Team ID")
	_ = cmd.MarkFlagRequired("team")
	return cmd
}

func newSprintStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "stop <id>",
		Short:       "Stop a sprint",
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

			teamSvc := team.New(store)
			term := terminal.NewCmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := sprint.New(teamSvc, store, term, ws, nil, cfg.Paths.Workspaces(), "")

			if err := svc.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"stopped": args[0]})
		},
	}
}

// resolveSprintContext resolves sprint ID and member ID from a flag value
// and CLIER_* environment variables. Member ID defaults to UserMemberID (human caller).
func resolveSprintContext(sprintFlag string) (sprintID, memberID string, err error) {
	sprintID = sprintFlag
	if sprintID == "" {
		sprintID = os.Getenv("CLIER_SPRINT_ID")
	}
	if sprintID == "" {
		return "", "", errors.New("--sprint flag or CLIER_SPRINT_ID must be set")
	}
	memberID = os.Getenv("CLIER_MEMBER_ID")
	if memberID == "" {
		memberID = domain.UserMemberID
	}
	return sprintID, memberID, nil
}

func newSprintListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sprints",
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

			sprints, err := store.ListSprints(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(sprints)
		},
	}
}
