package cmd

import (
	"errors"
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
		Short:   "Authenticate with GitHub",
		GroupID: rootGroupServer,
		Long:    `Authenticate with GitHub to access clier resources.`,
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
						Login: poll.User.Name,
					}
					if err := auth.Save(currentConfig().CredentialsPath, creds); err != nil {
						return fmt.Errorf("failed to save credentials: %w", err)
					}
					fmt.Fprintf(os.Stderr, "Logged in as %s\n", poll.User.Name)
					return nil
				}

				if poll.Status == "slow_down" {
					interval += 5 * time.Second
				}
			}

			return errors.New("login timed out — please try again")
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = client.Logout() // best-effort server-side logout
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
			creds, loadErr := auth.Load(currentConfig().CredentialsPath)
			if loadErr != nil {
				return printAuthLoggedOutStatus()
			}

			client := newAPIClient()
			user, userErr := client.GetCurrentUser()
			if userErr != nil {
				return printAuthExpiredStatus(creds.Login)
			}

			fmt.Fprintf(os.Stderr, "Logged in as %s\n", user.Name)
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
				return err
			}
			fmt.Println(creds.Token)
			return nil
		},
	}
}

func printAuthLoggedOutStatus() error {
	fmt.Fprintln(os.Stderr, "Not logged in.")
	return nil
}

func printAuthExpiredStatus(login string) error {
	fmt.Fprintf(os.Stderr, "Logged in as %s (token may be expired)\n", login)
	return nil
}
