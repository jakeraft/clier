package cmd

import (
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
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
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			var starredPtr *bool
			if cmd.Flags().Changed("starred") {
				starredPtr = &starred
			}
			opts := remoteapi.ListOptions{
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
				owner, err := currentLogin()
				if err != nil {
					return err
				}
				if owner == "" {
					return &domain.Fault{Kind: domain.KindLoginRequired}
				}
				items, err := svc.ListResources(owner, opts)
				if err != nil {
					return err
				}
				return present.Success(cmd.OutOrStdout(), view.ResourceListOf(items))
			}

			if len(args) == 1 {
				items, err := svc.ListResources(args[0], opts)
				if err != nil {
					return err
				}
				return present.Success(cmd.OutOrStdout(), view.ResourceListOf(items))
			}

			items, err := svc.ListPublicResources(opts)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceListOf(items))
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
