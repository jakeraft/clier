package cmd

import (
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRemoveCmd())
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <owner/name>",
		Short: "Remove a working copy and its run plans",
		Long: `Remove the working copy at <workspace_dir>/<owner>.<name>/
together with every run plan in <workspace_dir>/.runs/ that points
to it.

remove is the symmetric counterpart of clone — using it instead of
"rm -rf" keeps run lists clean and prevents orphan plans.

Refused when:
  - the working copy has uncommitted changes (push or revert first)
  - any associated run plan is still running (clier run stop first)`,
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
			base, err := workingCopyPath(owner, name)
			if err != nil {
				return err
			}
			fs := newFileMaterializer()
			svc, err := newWorkspaceOrchestratorWithFS(fs)
			if err != nil {
				return err
			}
			repo, err := newRunRepository()
			if err != nil {
				return err
			}
			removedRuns, err := svc.Remove(base, repo)
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}

			return present.Success(cmd.OutOrStdout(), view.RemoveResultOf(base, removedRuns))
		},
	}
}
