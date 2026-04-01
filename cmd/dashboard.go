package cmd

import (
	"fmt"
	"os/exec"

	"github.com/jakeraft/clier/internal/adapter/dashboard"
	"github.com/jakeraft/clier/web"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDashboardCmd())
}

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open a read-only dashboard snapshot in the browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			outPath, err := dashboard.Generate(cmd.Context(), store, cfg.Paths.Base(), web.DistFS, web.DistRoot)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Dashboard:", outPath)
			return exec.Command("open", outPath).Run() // macOS only
		},
	}
}
