package cmd

import (
	"fmt"
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
	var toMemberID string

	cmd := &cobra.Command{
		Use:   "send <content>",
		Short: "Send a message to a teammate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID := os.Getenv("CLIER_SPRINT_ID")
			fromMemberID := os.Getenv("CLIER_MEMBER_ID")
			if sprintID == "" || fromMemberID == "" {
				return fmt.Errorf("CLIER_SPRINT_ID and CLIER_MEMBER_ID must be set")
			}

			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewCmuxTerminal(store.DB())
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
	cmd.Flags().StringVar(&toMemberID, "to", "", "Recipient member ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}
