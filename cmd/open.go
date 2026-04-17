package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newOpenCmd())
}

func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open",
		Short:   "Open clier resources in a browser",
		GroupID: rootGroupSettings,
		RunE:    subcommandRequired,
	}
	cmd.AddCommand(newOpenDashboardCmd())
	return cmd
}

func newOpenDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open the dashboard in a browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := currentConfig()
			url := cfg.DashboardURL
			fmt.Printf("Opening %s\n", url)
			return openBrowser(url)
		},
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return &domain.Fault{
			Kind:    domain.KindUnsupportedPlatform,
			Subject: map[string]string{"platform": runtime.GOOS},
		}
	}
}
