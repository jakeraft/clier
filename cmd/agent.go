package cmd

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	for _, binary := range []domain.CliBinary{domain.BinaryClaude, domain.BinaryCodex} {
		rootCmd.AddCommand(newAgentCmd(binary))
	}
}

func newAgentCmd(binary domain.CliBinary) *cobra.Command {
	cmd := &cobra.Command{
		Use:   string(binary),
		Short: fmt.Sprintf("Manage %s CLI auth", binary),
	}

	cmd.AddCommand(newAgentCheckCmd(binary))
	return cmd
}

func isExitError(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func newAgentCheckCmd(binary domain.CliBinary) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: fmt.Sprintf("Check %s CLI auth status", binary),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if err := cfg.Auth.Check(binary); err != nil {
				if isExitError(err) {
					_, _ = fmt.Fprintf(w, "%s auth is invalid. Run: %s login\n", binary, binary)
				} else {
					_, _ = fmt.Fprintf(w, "%s is not logged in. Run: %s login\n", binary, binary)
				}
			} else {
				_, _ = fmt.Fprintf(w, "%s auth is valid\n", binary)
			}
			return nil
		},
	}
}
