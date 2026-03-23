package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/settings"
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
			s, err := settings.New()
			if err != nil {
				return err
			}
			return s.LoginAuth(binary)
		},
	}
}

func newAgentCheckCmd(binary domain.CliBinary) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: fmt.Sprintf("Check %s CLI auth status", binary),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := settings.New()
			if err != nil {
				return err
			}
			status, err := s.CheckAuth(binary)
			if err != nil {
				return err
			}
			switch status {
			case settings.AuthNotConfigured:
				fmt.Fprintf(cmd.OutOrStdout(), "%s auth not configured. Run: clier %s login\n", binary, binary)
			case settings.AuthInvalid:
				fmt.Fprintf(cmd.OutOrStdout(), "%s auth is invalid. Run: clier %s login\n", binary, binary)
			case settings.AuthOK:
				fmt.Fprintf(cmd.OutOrStdout(), "%s auth is valid\n", binary)
			}
			return nil
		},
	}
}
