package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/jakeraft/clier/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAuthCmd())
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "Log in to clier",
		GroupID: rootGroupServer,
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
			client := newAPIClient()

			resp, err := client.RequestDeviceCode()
			if err != nil {
				return fmt.Errorf("failed to start login: %w", err)
			}

			fmt.Fprintf(os.Stderr, "! First, copy your one-time code: %s\n", resp.UserCode)
			fmt.Fprintf(os.Stderr, "Then open: %s\n", resp.VerificationURI)
			fmt.Fprintf(os.Stderr, "Waiting for authentication...\n")

			interval := time.Duration(resp.Interval) * time.Second
			if interval == 0 {
				interval = 5 * time.Second
			}
			deadline := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

			for time.Now().Before(deadline) {
				time.Sleep(interval)

				poll, err := client.PollDeviceAuth(resp.DeviceCode)
				if err != nil {
					return fmt.Errorf("poll failed: %w", err)
				}

				if poll.AccessToken != "" && poll.User != nil {
					creds := &auth.Credentials{
						Token: poll.AccessToken,
						Login: poll.User.Login,
					}
					if err := auth.Save(currentConfig().CredentialsPath, creds); err != nil {
						return fmt.Errorf("failed to save credentials: %w", err)
					}
					fmt.Fprintf(os.Stderr, "Logged in as %s\n", poll.User.Login)
					return nil
				}

				if poll.Status == "slow_down" {
					interval += 5 * time.Second
				}
			}

			return fmt.Errorf("login timed out — please try again")
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Delete(currentConfig().CredentialsPath); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Logged out.")
			return nil
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show login status",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := auth.Load(currentConfig().CredentialsPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Not logged in.")
				return nil
			}

			client := newAPIClient()
			user, err := client.GetCurrentUser()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Logged in as %s (token may be expired)\n", creds.Login)
				return nil
			}

			fmt.Fprintf(os.Stderr, "Logged in as %s\n", user.Login)
			return nil
		},
	}
}

func newAuthTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the current access token",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := auth.Load(currentConfig().CredentialsPath)
			if err != nil {
				return fmt.Errorf("not logged in. Run 'clier auth login' first.")
			}
			fmt.Println(creds.Token)
			return nil
		},
	}
}
