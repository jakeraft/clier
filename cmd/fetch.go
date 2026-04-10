package cmd

import (
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newFetchCmd())
}

func newFetchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fetch",
		Short:   "Fetch external updates without changing the local clone",
		GroupID: rootGroupRuntime,
	}
	cmd.AddCommand(newFetchUpstreamCmd())
	return cmd
}

func newFetchUpstreamCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upstream",
		Short: "Fetch the current fork's upstream snapshot",
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
			result, err := svc.FetchUpstream(copyRoot)
			if err != nil {
				return err
			}
			return printJSON(result)
		},
	}
}
