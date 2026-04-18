package cmd

import (
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newPushCmd())
}

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push <owner/name>",
		Short: "Push tracked local changes for a working copy",
		Long: `Push locally modified resources to the server for the working copy
at <workspace_dir>/<owner>.<name>/. Only resources that have changed
since the last pull/clone are sent. Fails if the remote version has
changed (pull first to resolve).`,
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
			result, err := svc.Push(base)
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}
			return present.Success(cmd.OutOrStdout(), view.PushResultOf(result))
		},
	}
}
