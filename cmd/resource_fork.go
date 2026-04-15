package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCopyCmd())
}

func newCopyCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "copy <owner/name>",
		Short:   "Copy a resource into your namespace",
		Long:    `Copy a resource into your namespace. Creates your own copy that you can customize independently.`,
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

			resp, err := client.CopyResource(kind, owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}
