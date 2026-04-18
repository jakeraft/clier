package cmd

import (
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newStatusCmd())
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <owner/name>",
		Short: "Show a working copy's status",
		Long: `Show the status of a working copy at <workspace_dir>/<owner>.<name>/.

Displays which tracked resources have local modifications and any
runs spawned from this working copy.`,
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
			runPlansDir, err := runsDir()
			if err != nil {
				return err
			}
			svc, err := newWorkspaceOrchestrator()
			if err != nil {
				return err
			}
			status, err := svc.Status(base, runPlansDir)
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}
			return present.Success(cmd.OutOrStdout(), view.StatusResultOf(status))
		},
	}
}
