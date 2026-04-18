package cmd

import (
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newFetchCmd())
}

func newFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <owner/name>",
		Short: "Preview remote changes for a working copy",
		Long: `Preview which tracked resources would change if you pulled the
latest version of a working copy at <workspace_dir>/<owner>.<name>/.

This command compares against the latest remote team state without
writing local files.`,
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
			svc, err := newWorkspaceOrchestrator()
			if err != nil {
				return err
			}
			result, err := svc.Fetch(base)
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}
			return present.Success(cmd.OutOrStdout(), view.FetchResultOf(base, result))
		},
	}
}
