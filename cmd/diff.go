package cmd

import (
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDiffCmd())
}

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "diff",
		Short:   "Show differences against fetched snapshots",
		GroupID: rootGroupWorkspace,
	}
	cmd.AddCommand(newDiffUpstreamCmd())
	return cmd
}

func newDiffUpstreamCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upstream",
		Short: "Show differences between local and fetched upstream",
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
			result, err := svc.DiffFetchedUpstream(copyRoot)
			if err != nil {
				return err
			}
			return printJSON(result)
		},
	}
}
