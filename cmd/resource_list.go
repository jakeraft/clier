package cmd

import (
	"errors"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newListCmd())
}

func newListCmd() *cobra.Command {
	var kind, query string
	var mine bool
	var limit, offset int

	cmd := &cobra.Command{
		Use:     "list [owner]",
		Short:   "List resources",
		Long:    `List public resources, or a specific owner's resources.`,
		GroupID: rootGroupResources,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			opts := api.ListOptions{
				Kind:   kind,
				Query:  query,
				Limit:  limit,
				Offset: offset,
			}

			if mine {
				owner := currentLogin()
				if owner == "" {
					return errors.New("--mine requires login: run 'clier auth login'")
				}
				items, err := client.ListResources(owner, opts)
				if err != nil {
					return err
				}
				return printJSON(items)
			}

			if len(args) == 1 {
				items, err := client.ListResources(args[0], opts)
				if err != nil {
					return err
				}
				return printJSON(items)
			}

			items, err := client.ListPublicResources(opts)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Filter by resource kind (member, team, skill, claude-md, claude-setting, codex-md, codex-setting)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query")
	cmd.Flags().BoolVar(&mine, "mine", false, "List only my resources")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
	return cmd
}
