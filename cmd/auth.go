package cmd

import (
	"fmt"
	"os"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	"github.com/jakeraft/clier/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAuthCmd())
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "Log in and manage credentials",
		GroupID: rootGroupSettings,
		Long:    `Log in and manage credentials for accessing clier resources.`,
		RunE:    subcommandRequired,
	}
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthTokenCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in with GitHub",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteAuthService()
			if err != nil {
				return err
			}
			cfg, err := currentConfig()
			if err != nil {
				return err
			}
			user, err := svc.Login(cfg.CredentialsPath, func(prompt auth.LoginPrompt) {
				fmt.Fprintf(os.Stderr, "! First, copy your one-time code: %s\n", prompt.UserCode)
				fmt.Fprintf(os.Stderr, "Then open: %s\n", prompt.VerificationURI)
				fmt.Fprintf(os.Stderr, "Waiting for authentication...\n")
			})
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.AuthLoginOf(user.Name))
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteAuthService()
			if err != nil {
				return err
			}
			cfg, err := currentConfig()
			if err != nil {
				return err
			}
			if err := svc.Logout(cfg.CredentialsPath); err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.AuthLogoutOf())
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show login status",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteAuthService()
			if err != nil {
				return err
			}
			cfg, err := currentConfig()
			if err != nil {
				return err
			}
			user, err := svc.Status(cfg.CredentialsPath)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.AuthStatusOf(user.Name))
		},
	}
}

func newAuthTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the current access token",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteAuthService()
			if err != nil {
				return err
			}
			cfg, err := currentConfig()
			if err != nil {
				return err
			}
			token, err := svc.Token(cfg.CredentialsPath)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.AuthTokenOf(token))
		},
	}
}
