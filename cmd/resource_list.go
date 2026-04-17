package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newListCmd())
}

func newListCmd() *cobra.Command {
	var kind, query, uses, sort, order string
	var mine, starred bool
	var limit, offset int

	cmd := &cobra.Command{
		Use:     "list [owner]",
		Short:   "List resources",
		Long:    `List public resources, or a specific owner's resources.`,
		GroupID: rootGroupResources,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			var starredPtr *bool
			if cmd.Flags().Changed("starred") {
				starredPtr = &starred
			}
			opts := api.ListOptions{
				Kind:    kind,
				Query:   query,
				Uses:    uses,
				Starred: starredPtr,
				Limit:   limit,
				Offset:  offset,
				Sort:    sort,
				Order:   order,
			}

			if mine {
				owner := currentLogin()
				if owner == "" {
					return &domain.Fault{Kind: domain.KindLoginRequired}
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
	cmd.Flags().StringVar(&kind, "kind", "", "Filter by resource kind (team, skill, instruction, claude-setting, codex-setting)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query")
	cmd.Flags().BoolVar(&mine, "mine", false, "List only my resources")
	cmd.Flags().StringVar(&uses, "uses", "", "Filter: resources that reference this target (owner/name)")
	cmd.Flags().BoolVar(&starred, "starred", false, "Filter: only starred resources")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort field (updated_at, star_count)")
	cmd.Flags().StringVar(&order, "order", "", "Sort order (asc, desc)")
	return cmd
}
