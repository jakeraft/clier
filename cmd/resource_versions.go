package cmd

import "github.com/spf13/cobra"

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
			client := newAPIClient()
			owner, name, err := parseExplicitOwnerName(args[0])
			if err != nil {
				return err
			}
			items, err := client.ListResourceVersions(owner, name)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}
