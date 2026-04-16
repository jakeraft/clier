package cmd

import (
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newPullCmd())
}

func newPullCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull latest changes from the server",
		Long: `Pull the latest version of tracked resources from the server,
updating local projections and materialized files. Fails if you
have local modifications unless --force is used.`,
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

			svc := appworkspace.NewService(newAPIClient(), fs, git)
			manifest, err := svc.Pull(copyRoot, force)
			if err != nil {
				return err
			}
			return printJSON(map[string]any{
				"status": "pulled",
				"kind":   manifest.Kind,
				"owner":  manifest.Owner,
				"name":   manifest.Name,
				"state":  appworkspace.ManifestPath(copyRoot),
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite local changes in tracked files")
	return cmd
}
