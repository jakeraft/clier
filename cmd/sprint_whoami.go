package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/app/sprint"
	"github.com/jakeraft/clier/internal/app/team"
	"github.com/spf13/cobra"
)

func newSprintWhoamiCmd() *cobra.Command {
	var sprintFlag string

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current member's position in the sprint team",
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID, memberID, err := resolveSprintContext(sprintFlag)
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

			teamSvc := team.New(store)
			term := terminal.NewCmuxTerminal(store)
			svc := sprint.New(teamSvc, store, term, nil, "")

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
