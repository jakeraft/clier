package cmd

import (
	"fmt"

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
		Use:         "create",
		Short:       "Create a team",
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			member, err := store.GetMember(cmd.Context(), rootMemberID)
			if err != nil {
				return fmt.Errorf("get root member: %w", err)
			}
			t, err := domain.NewTeam(name, rootMemberID, member.Name)
			if err != nil {
				return err
			}
			if err := store.CreateTeam(cmd.Context(), t); err != nil {
				return err
			}
			return printJSON(t)
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
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
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
		Use:         "update <id>",
		Short:       "Update a team",
		Annotations: map[string]string{mutates: "true"},
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
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
		Use:         "delete <id>",
		Short:       "Delete a team",
		Annotations: map[string]string{mutates: "true"},
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
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
		Use:         "add <team-id> <member-id>",
		Short:       "Add a member to a team",
		Annotations: map[string]string{mutates: "true"},
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID, memberID := args[0], args[1]

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			member, err := store.GetMember(cmd.Context(), memberID)
			if err != nil {
				return fmt.Errorf("get member: %w", err)
			}
			t, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			tm, err := t.AddTeamMember(memberID, member.Name)
			if err != nil {
				return err
			}
			if err := store.AddTeamMember(cmd.Context(), teamID, *tm); err != nil {
				return err
			}
			return printJSON(t)
		},
	}
}

func newTeamMemberRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "remove <team-id> <team-member-id>",
		Short:       "Remove a team member from a team",
		Annotations: map[string]string{mutates: "true"},
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID, teamMemberID := args[0], args[1]

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			t, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			if err := t.RemoveTeamMember(teamMemberID); err != nil {
				return err
			}
			if err := store.RemoveTeamMember(cmd.Context(), teamID, teamMemberID); err != nil {
				return err
			}
			return printJSON(t)
		},
	}
}

func newTeamMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <team-id>",
		Short: "List members of a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printJSON(team.TeamMembers)
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
	var from, to string

	cmd := &cobra.Command{
		Use:         "add <team-id>",
		Short:       "Add a relation to a team",
		Annotations: map[string]string{mutates: "true"},
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			r := domain.Relation{From: from, To: to}
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
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newTeamRelationRemoveCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:         "remove <team-id>",
		Short:       "Remove a relation from a team",
		Annotations: map[string]string{mutates: "true"},
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			team, err := store.GetTeam(cmd.Context(), teamID)
			if err != nil {
				return err
			}
			if err := team.RemoveRelation(from, to); err != nil {
				return err
			}
			if err := store.RemoveTeamRelation(cmd.Context(), teamID, domain.Relation{From: from, To: to}); err != nil {
				return err
			}
			return printJSON(team)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "From member ID")
	cmd.Flags().StringVar(&to, "to", "", "To member ID")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newTeamRelationListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <team-id>",
		Short: "List relations of a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
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

