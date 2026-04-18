package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newForkCmd())
}

func newForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <owner/name>",
		Short: "Fork a resource to customize it",
		Long: `Create a server-side fork of another owner's resource.
The copy lives in your namespace and can be edited independently.
Not required for running — use clone to run any resource directly.`,
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

			// Detect kind via GET.
			res, err := svc.GetResource(owner, name)
			if err != nil {
				return fmt.Errorf("look up resource %q: %w", args[0], err)
			}
			kind := remoteapi.ResourceKind(res.Kind)

			resp, err := svc.ForkResource(kind, owner, name)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))
		},
	}
}
