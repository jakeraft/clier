package cmd

import (
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newPullCmd())
}

func newPullCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "pull <owner/name>",
		Short: "Pull latest changes for a working copy",
		Long: `Pull the latest version of tracked resources for a working copy at
<workspace_dir>/<owner>.<name>/, updating local projections and
materialized files. Fails if local modifications exist unless
--force is used.`,
		GroupID: rootGroupWorkspace,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}
			if err := validateOwner(owner); err != nil {
				return err
			}
			base := workingCopyPath(owner, name)

			svc := appworkspace.NewService(newAPIClient(), newFileMaterializer(), newGitRepo())
			manifest, err := svc.Pull(base, force)
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}
			return printJSON(map[string]any{
				"status": "pulled",
				"kind":   manifest.Kind,
				"owner":  manifest.Owner,
				"name":   manifest.Name,
				"state":  appworkspace.ManifestPath(base),
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite local changes in tracked files")
	return cmd
}
