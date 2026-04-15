package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDeleteCmd())
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <[owner/]name>",
		Short:   "Delete a resource (auto-detects kind)",
		GroupID: rootGroupResources,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, err := parseOwnerName(args[0])
			if err != nil {
				return err
			}

			// Detect kind via GET.
			res, err := client.GetResource(owner, name)
			if err != nil {
				return fmt.Errorf("look up resource %q: %w", args[0], err)
			}
			kind := api.ResourceKind(res.Kind)

			if err := client.DeleteResource(kind, owner, name); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
