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

Use list, view, create, edit, and delete to manage your
team definitions.

Workflow:
  clier team create        Define a new team
  clier clone <name>       Clone it to your machine
  clier run start          Start the current local clone`,
	}
	cmd.AddGroup(&cobra.Group{ID: subGroupServer, Title: "Define"})
	cmd.AddCommand(newTeamListCmd())
	cmd.AddCommand(newTeamViewCmd())
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamEditCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	return cmd
}

func newTeamListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [owner]",
		Short:   "List your teams",
		Long:    "List your teams, or another user's teams if [owner] is given.",
		GroupID: subGroupServer,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			var owner string
			if len(args) == 1 {
				owner = args[0]
			} else {
				owner = requireLogin()
			}
			teams, err := client.ListTeams(owner)
			if err != nil {
				return err
			}
			return printJSON(teams)
		},
	}
}

func newTeamViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <[owner/]name>",
		Short:   "Show team details",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			team, err := client.GetTeam(owner, name)
			if err != nil {
				return err
			}
			return printJSON(team)
		},
	}
}

func newTeamCreateCmd() *cobra.Command {
	var name string
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
			body := api.TeamMutationRequest{
				Name:        name,
				TeamMembers: members,
				Relations:   parsedRelations,
				RootIndex:   &rootIndex,
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
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("member")
	_ = cmd.MarkFlagRequired("root-index")
	return cmd
}

func newTeamEditCmd() *cobra.Command {
	var name string
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
			current, err := client.GetTeam(owner, args[0])
			if err != nil {
				return err
			}
			body, err := teamMutationRequestFromResponse(current)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				body.Name = name
			}
			if cmd.Flags().Changed("member") {
				body.TeamMembers, err = parseTeamMemberSpecs(teamMembers)
				if err != nil {
					return err
				}
				if !cmd.Flags().Changed("relation") {
					body.Relations = nil
				}
				if !cmd.Flags().Changed("root-index") {
					return errors.New("--root-index is required when replacing --member because team membership is index-based")
				}
			}
			if cmd.Flags().Changed("relation") {
				body.Relations, err = parseTeamRelationSpecs(relations)
				if err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("root-index") {
				if rootIndex < 0 {
					body.RootIndex = nil
				} else {
					body.RootIndex = &rootIndex
				}
			}
			resp, err := client.UpdateTeam(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New team name")
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
