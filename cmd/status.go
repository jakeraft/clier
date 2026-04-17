package cmd

import (
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newStatusCmd())
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <owner/name>",
		Short: "Show a working copy's status",
		Long: `Show the status of a working copy at <workspace_dir>/<owner>/<name>/.

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
			base := workingCopyPath(owner, name)

			svc := appworkspace.NewService(newAPIClient(), newFileMaterializer(), newGitRepo())
			status, err := svc.Status(base, runsDir())
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}
			return printJSON(status)
		},
	}
}
