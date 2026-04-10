package cmd

import (
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newMergeCmd())
}

func newMergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge",
		Short:   "Merge fetched snapshots into the local clone",
		GroupID: rootGroupRuntime,
	}
	cmd.AddCommand(newMergeUpstreamCmd())
	return cmd
}

func newMergeUpstreamCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upstream",
		Short: "Merge the fetched upstream snapshot into the root projection",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := resolveCurrentDir()
			if err != nil {
				return err
			}
			copyRoot, _, err := appworkspace.FindManifestAbove(base)
			if err != nil {
				if os.IsNotExist(err) {
					return errNotInWorkingCopy()
				}
				return err
			}

			svc := appworkspace.NewService(newAPIClient())
			result, err := svc.MergeFetchedUpstream(copyRoot)
			if err != nil {
				return err
			}
			return printJSON(result)
		},
	}
}
