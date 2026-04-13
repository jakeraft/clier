package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTeamCmd())
}

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "team",
		Short:   "Manage agent teams",
		GroupID: rootGroupServer,
		Long:    `Create, edit, and delete agent team compositions on the server.`,
	}
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamEditCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	return cmd
}

func newTeamCreateCmd() *cobra.Command {
	var name, summary string
	var teamMembers, relations []string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new team",

		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			members, err := parseTeamMemberSpecs(teamMembers)
			if err != nil {
				return err
			}
			parsedRelations, err := parseTeamRelationSpecs(relations)
			if err != nil {
				return err
			}
			body := api.TeamWriteRequest{
				Name:        name,
				TeamMembers: members,
				Relations:   parsedRelations,
				Summary:     summary,
			}
			resp, err := client.CreateResource(api.KindTeam, owner, body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Team member as <member-id>@<version>; repeat for each member")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Relation as <from-member-id>:<to-member-id>; repeat for each edge")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("member")
	return cmd
}

func newTeamEditCmd() *cobra.Command {
	var name, summary string
	var teamMembers, relations []string

	cmd := &cobra.Command{
		Use:     "edit <name>",
		Short:   "Update a team",

		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			body := api.TeamPatchRequest{}
			if cmd.Flags().Changed("name") {
				body.Name = &name
			}
			if cmd.Flags().Changed("summary") {
				body.Summary = &summary
			}
			if cmd.Flags().Changed("member") {
				members, err := parseTeamMemberSpecs(teamMembers)
				if err != nil {
					return err
				}
				body.TeamMembers = members
				if !cmd.Flags().Changed("relation") {
					body.Relations = []api.TeamRelationRequest{}
				}
			}
			if cmd.Flags().Changed("relation") {
				parsed, err := parseTeamRelationSpecs(relations)
				if err != nil {
					return err
				}
				body.Relations = parsed
			}
			resp, err := client.PatchResource(api.KindTeam, owner, args[0], &body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New team name")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Replace team members with <member-id>@<version>; repeat for each member")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Replace relations with <from-member-id>:<to-member-id>; repeat for each edge")
	return cmd
}

func newTeamDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a team",

		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteResource(api.KindTeam, owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
