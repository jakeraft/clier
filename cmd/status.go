package cmd

import (
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newStatusCmd())
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Show the current local clone status",
		GroupID: rootGroupRuntime,
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := resolveCurrentDir()
			if err != nil {
				return err
			}
			copyRoot, _, err := appworkspace.FindManifestAbove(newFileMaterializer(), base)
			if err != nil {
				if os.IsNotExist(err) {
					return errNotInWorkingCopy()
				}
				return err
			}

			svc := appworkspace.NewService(newAPIClient(), newFileMaterializer(), newGitRepo())
			status, err := svc.Status(copyRoot)
			if err != nil {
				return err
			}
			return printJSON(status)
		},
	}
}
