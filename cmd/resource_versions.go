package cmd

import (
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newVersionsCmd())
}

func newVersionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "versions <owner/name>",
		Short:   "List versions of a resource",
		GroupID: rootGroupResources,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}
			items, err := svc.ListResourceVersions(owner, name)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ItemsOf(items))
		},
	}
}
