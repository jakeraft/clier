package cmd

import (
	"errors"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTeamCmd())
}

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "team",
		Short:   "Compose agent teams",
		GroupID: rootGroupServer,
		Long: `Compose agent teams on the server.

Use create, edit, and delete to manage your
own team definitions.

Workflow:
  clier team create        Define a new team
  clier explore team <owner/name>
                           Inspect an existing team
  clier clone <name>       Clone your team to your machine
  clier run start          Start the current local clone`,
	}
	cmd.AddGroup(&cobra.Group{ID: subGroupServer, Title: "Define"})
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamEditCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	return cmd
}

func newTeamCreateCmd() *cobra.Command {
	var name, summary string
	var teamMembers, relations []string
	rootIndex := -1

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new team",
		GroupID: subGroupServer,
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
			if rootIndex < 0 {
				return errors.New("--root-index must be set to a non-negative team_members index")
			}
			body := api.TeamWriteRequest{
				Name:        name,
				TeamMembers: members,
				Relations:   parsedRelations,
				RootIndex:   &rootIndex,
				Summary:     summary,
			}
			resp, err := client.CreateTeam(owner, body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Team member as <member-id>@<version>:<name>; repeat for each member")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Relation as <from-index>:<to-index> using zero-based --member indices; repeat for each edge")
	cmd.Flags().IntVar(&rootIndex, "root-index", -1, "Root member index in the zero-based --member list")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("member")
	_ = cmd.MarkFlagRequired("root-index")
	return cmd
}

func newTeamEditCmd() *cobra.Command {
	var name, summary string
	var teamMembers, relations []string
	rootIndex := -1

	cmd := &cobra.Command{
		Use:     "edit <name>",
		Short:   "Update a team",
		GroupID: subGroupServer,
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
				if !cmd.Flags().Changed("root-index") {
					return errors.New("--root-index is required when replacing --member because team membership is index-based")
				}
			}
			if cmd.Flags().Changed("relation") {
				parsed, err := parseTeamRelationSpecs(relations)
				if err != nil {
					return err
				}
				body.Relations = parsed
			}
			if cmd.Flags().Changed("root-index") {
				if rootIndex < 0 {
					body.RootIndex = nil
				} else {
					body.RootIndex = &rootIndex
				}
			}
			resp, err := client.PatchTeam(owner, args[0], &body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New team name")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Replace team members with <member-id>:<name>; repeat for each member")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Replace relations with <from-index>:<to-index> using zero-based member indices; repeat for each edge")
	cmd.Flags().IntVar(&rootIndex, "root-index", -1, "Replace root member index; use -1 to clear")
	return cmd
}

func newTeamDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a team",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteTeam(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
