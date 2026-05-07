package cmd

import (
	"fmt"
	"os"
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
		Args:  cobra.ArbitraryArgs,
		RunE:  helpOrUnknown,
	}
	cmd.AddCommand(newOpenDashboardCmd())
	return cmd
}

func newOpenDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open the dashboard URL in your default browser",
		Args:  cobra.NoArgs,
		Long: `Open the configured dashboard URL with the OS-native browser
launcher.

The default URL is baked at build time. Override per-invocation
with the CLIER_DASHBOARD_URL environment variable. Run
` + "`clier version`" + ` to see the active URL.`,
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
	// Detach — the CLI returns immediately after the launcher has
	// spawned the URL handler. Reap the child in the background so the
	// process table stays clean; surface a non-zero exit (no handler,
	// permission denied, …) on stderr because the user already moved on
	// and never sees a return value.
	go func() {
		if werr := cmd.Wait(); werr != nil {
			fmt.Fprintf(os.Stderr, "warning: browser launcher exited with: %s\n", werr)
		}
	}()
	return nil
}
