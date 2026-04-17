package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
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
					// Server may return an error for authorization_pending (RFC 8628).
					// Continue polling unless it's a non-retryable error.
					var apiErr *api.Error
					if errors.As(err, &apiErr) && apiErr.StatusCode < 500 {
						continue
					}
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
		Use:          "status",
		Short:        "Show login status",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, _ := auth.Load(currentConfig().CredentialsPath)
			var (
				user    *api.UserResponse
				userErr error
			)
			if creds != nil {
				user, userErr = newAPIClient().GetCurrentUser()
			}
			msg, ok := authStatusResult(creds, user, userErr)
			fmt.Fprintln(os.Stderr, msg)
			if !ok {
				return errSilent
			}
			return nil
		},
	}
}

// authStatusResult reports the login state based on the stored credentials
// and the server's response to a verification request. The boolean is true
// only when the server has confirmed the token is valid.
func authStatusResult(creds *auth.Credentials, user *api.UserResponse, userErr error) (string, bool) {
	if creds == nil {
		return "Not logged in.", false
	}
	if userErr == nil && user != nil {
		return fmt.Sprintf("Logged in as %s", user.Name), true
	}
	var apiErr *api.Error
	if errors.As(userErr, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden) {
		return "Not logged in: stored token is invalid or expired.\nRun 'clier auth login' to re-authenticate.", false
	}
	return fmt.Sprintf("Unable to verify login for %s: %v", creds.Login, userErr), false
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

