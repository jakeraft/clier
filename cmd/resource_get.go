package cmd

import (
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newGetCmd())
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <owner/name[@version]>",
		Short:   "Show any resource by owner and name",
		GroupID: rootGroupResources,
		Args:    requireOneArg("clier get <owner/name[@version]>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, name, version, err := splitVersionedResourceID(args[0])
			if err != nil {
				return err
			}
			if version != nil {
				item, err := svc.GetResourceVersion(owner, name, *version)
				if err != nil {
					return err
				}
				return present.Success(cmd.OutOrStdout(), view.ResourceVersionOf(item))
			}
			item, err := svc.GetResource(owner, name)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceOf(item))
		},
	}
}
