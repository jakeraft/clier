package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newOpenCmd())
}

func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open clier surfaces in your browser",
		RunE: func(c *cobra.Command, _ []string) error {
			return c.Help()
		},
	}
	cmd.AddCommand(newOpenDashboardCmd())
	return cmd
}

func newOpenDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open the dashboard URL in your default browser",
		Long: `Open the configured dashboard URL with the OS-native browser
launcher.

Override via the CLIER_DASHBOARD_URL environment variable; default is
http://localhost:5173 (local-dev).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := openBrowser(cfg.DashboardURL); err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"opened": cfg.DashboardURL,
			})
		},
	}
}

// openBrowser launches the OS-native URL handler. macOS uses `open`,
// Linux uses `xdg-open`, Windows uses `rundll32 url.dll,FileProtocolHandler`
// — these are the well-trodden cross-platform invocations and avoid
// pulling in a browser-launch dependency.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("open dashboard: unsupported OS %s — visit %s manually", runtime.GOOS, url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	// Detach — we don't wait for the browser process to exit. The CLI
	// returns immediately after the launcher has spawned the URL handler.
	go func() { _ = cmd.Wait() }()
	return nil
}
