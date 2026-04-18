package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDeleteCmd())
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <owner/name>",
		Short:   "Delete a resource (auto-detects kind)",
		GroupID: rootGroupResources,
		Args:    requireOneArg("clier delete <owner/name>"),
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

			if err := svc.DeleteResource(kind, owner, name); err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.DeletedOf(args[0]))
		},
	}
}
