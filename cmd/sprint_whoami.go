package cmd

import (
	"errors"
	"os"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/app/sprint"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func newSprintWhoamiCmd() *cobra.Command {
	var sprintFlag string

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current member's position in the sprint team",
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID := sprintFlag
			if sprintID == "" {
				sprintID = os.Getenv("CLIER_SPRINT_ID")
			}
			memberID := os.Getenv("CLIER_MEMBER_ID")
			if memberID == "" {
				memberID = domain.UserMemberID
			}
			if sprintID == "" {
				return errors.New("--sprint flag or CLIER_SPRINT_ID must be set")
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
			svc := sprint.New(store, term, nil, cfg.Paths.Base())

			result, err := svc.Whoami(cmd.Context(), sprintID, memberID)
			if err != nil {
				return err
			}
			return printJSON(result)
		},
	}
	cmd.Flags().StringVar(&sprintFlag, "sprint", "", "Sprint ID (defaults to CLIER_SPRINT_ID)")
	return cmd
}
