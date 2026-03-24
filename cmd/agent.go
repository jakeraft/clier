package cmd

import (
	"errors"
	"fmt"
	"os"
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

	cmd.AddCommand(newAgentLoginCmd(binary))
	cmd.AddCommand(newAgentCheckCmd(binary))
	return cmd
}

func newAgentLoginCmd(binary domain.CliBinary) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: fmt.Sprintf("Login to %s CLI", binary),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSettings()
			if err != nil {
				return err
			}
			return s.LoginAuth(binary)
		},
	}
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
			s, err := newSettings()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if err := s.CheckAuth(binary); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					_, _ = fmt.Fprintf(w, "%s auth not configured. Run: clier %s login\n", binary, binary)
				} else if isExitError(err) {
					_, _ = fmt.Fprintf(w, "%s auth is invalid. Run: clier %s login\n", binary, binary)
				} else {
					return err
				}
			} else {
				_, _ = fmt.Fprintf(w, "%s auth is valid\n", binary)
			}
			return nil
		},
	}
}
