package cmd

import (
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTeamCmd())
}

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage teams",
	}
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamListCmd())
	cmd.AddCommand(newTeamUpdateCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	cmd.AddCommand(newTeamMemberCmd())
	cmd.AddCommand(newTeamRelationCmd())
	return cmd
}

func newTeamCreateCmd() *cobra.Command {
	var name, rootMemberID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := domain.NewTeam(name, rootMemberID)
			if err != nil {
				return err
			}
			if err := store.CreateTeam(cmd.Context(), team); err != nil {
				return err
			}
			return printJSON(team)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringVar(&rootMemberID, "root-member", "", "Root member ID")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("root-member")
	return cmd
}

func newTeamListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			teams, err := store.ListTeams(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(teams)
		},
	}
}

func newTeamUpdateCmd() *cobra.Command {
	var name, rootMemberID string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var rootMemberPtr *string
			if cmd.Flags().Changed("root-member") {
				rootMemberPtr = &rootMemberID
			}

			if err := team.Update(namePtr, rootMemberPtr); err != nil {
				return err
			}
			if err := store.UpdateTeam(cmd.Context(), &team); err != nil {
				return err
			}
			return printJSON(team)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New team name")
	cmd.Flags().StringVar(&rootMemberID, "root-member", "", "New root member ID")
	return cmd
}

func newTeamDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.DeleteTeam(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newTeamMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage team members",
	}
	cmd.AddCommand(newTeamMemberAddCmd())
	cmd.AddCommand(newTeamMemberRemoveCmd())
	cmd.AddCommand(newTeamMemberListCmd())
	return cmd
}

func newTeamMemberAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <team-id> <member-id>",
		Short: "Add a member to a team",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID, memberID := args[0], args[1]

			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			if err := team.AddMember(memberID); err != nil {
				return err
			}
			if err := store.AddTeamMember(cmd.Context(), teamID, memberID); err != nil {
				return err
			}
			return printJSON(team)
		},
	}
}

func newTeamMemberRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <team-id> <member-id>",
		Short: "Remove a member from a team",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID, memberID := args[0], args[1]

			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			if err := team.RemoveMember(memberID); err != nil {
				return err
			}
			if err := store.RemoveTeamMember(cmd.Context(), teamID, memberID); err != nil {
				return err
			}
			return printJSON(team)
		},
	}
}

func newTeamMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <team-id>",
		Short: "List members of a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printJSON(team.MemberIDs)
		},
	}
}

func newTeamRelationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relation",
		Short: "Manage team relations",
	}
	cmd.AddCommand(newTeamRelationAddCmd())
	cmd.AddCommand(newTeamRelationRemoveCmd())
	cmd.AddCommand(newTeamRelationListCmd())
	return cmd
}

func newTeamRelationAddCmd() *cobra.Command {
	var from, to, relType string

	cmd := &cobra.Command{
		Use:   "add <team-id>",
		Short: "Add a relation to a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			r := domain.Relation{From: from, To: to, Type: domain.RelationType(relType)}
			if err := team.AddRelation(r); err != nil {
				return err
			}
			if err := store.AddTeamRelation(cmd.Context(), teamID, r); err != nil {
				return err
			}
			return printJSON(team)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "From member ID")
	cmd.Flags().StringVar(&to, "to", "", "To member ID")
	cmd.Flags().StringVar(&relType, "type", "", "Relation type (leader|peer)")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func newTeamRelationRemoveCmd() *cobra.Command {
	var from, to, relType string

	cmd := &cobra.Command{
		Use:   "remove <team-id>",
		Short: "Remove a relation from a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			if err := team.RemoveRelation(from, to, domain.RelationType(relType)); err != nil {
				return err
			}
			if err := store.RemoveTeamRelation(cmd.Context(), teamID, domain.Relation{From: from, To: to, Type: domain.RelationType(relType)}); err != nil {
				return err
			}
			return printJSON(team)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "From member ID")
	cmd.Flags().StringVar(&to, "to", "", "To member ID")
	cmd.Flags().StringVar(&relType, "type", "", "Relation type (leader|peer)")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func newTeamRelationListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <team-id>",
		Short: "List relations of a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printJSON(team.Relations)
		},
	}
}
