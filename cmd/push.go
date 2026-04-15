package cmd

import (
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newPushCmd())
}

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push tracked local changes to the server",
		Long: `Push locally modified resources to the server. Only resources
that have changed since the last pull/clone are sent. Fails if
the remote version has changed (pull first to resolve).`,
		GroupID: rootGroupWorkspace,
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := resolveCurrentDir()
			if err != nil {
				return err
			}
			fs := newFileMaterializer()
			git := newGitRepo()
			copyRoot, _, err := appworkspace.FindManifestAbove(fs, base)
			if err != nil {
				if os.IsNotExist(err) {
					return errNotInWorkingCopy()
				}
				return err
			}

			login := requireLogin()
			svc := appworkspace.NewService(newAPIClient(), fs, git)
			result, err := svc.Push(copyRoot, login)
			if err != nil {
				return err
			}
			return printJSON(result)
		},
	}
}
