package cmd

import (
	"errors"
	"os"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/app/sprint"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newMessageCmd())
}

func newMessageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "message",
		Short: "Manage messages",
	}
	cmd.AddCommand(newMessageSendCmd())
	return cmd
}

func newMessageSendCmd() *cobra.Command {
	var sprintFlag, toMemberID string

	cmd := &cobra.Command{
		Use:   "send <content>",
		Short: "Send a message to a teammate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID := sprintFlag
			if sprintID == "" {
				sprintID = os.Getenv("CLIER_SPRINT_ID")
			}
			fromMemberID := os.Getenv("CLIER_MEMBER_ID")
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
			svc := sprint.New(store, term, nil)

			if err := svc.DeliverMessage(cmd.Context(), sprintID, fromMemberID, toMemberID, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "delivered",
				"from":   fromMemberID,
				"to":     toMemberID,
			})
		},
	}
	cmd.Flags().StringVar(&sprintFlag, "sprint", "", "Sprint ID (defaults to CLIER_SPRINT_ID)")
	cmd.Flags().StringVar(&toMemberID, "to", "", "Recipient member ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}
