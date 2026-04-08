package cmd

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Manage Claude CLI auth",
	}
	cmd.AddCommand(newAgentCheckCmd())
	rootCmd.AddCommand(cmd)
}

func isExitError(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func newAgentCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check Claude CLI auth status",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			check := exec.Command("claude", "auth", "status")
			if err := check.Run(); err != nil {
				if isExitError(err) {
					_, _ = fmt.Fprintf(w, "claude auth is invalid. Run: claude login\n")
				} else {
					_, _ = fmt.Fprintf(w, "claude is not logged in. Run: claude login\n")
				}
			} else {
				_, _ = fmt.Fprintf(w, "claude auth is valid\n")
			}
			return nil
		},
	}
}
