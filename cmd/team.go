package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/app/team"
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
	cmd.AddCommand(newTeamExportCmd())
	cmd.AddCommand(newTeamImportCmd())
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
		Use:         "remove <team-id> <member-id>",
		Short:       "Remove a member from a team",
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

func newTeamExportCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "export <team-id>",
		Short: "Export a team and all sub-resources to JSON",
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

			svc := team.New(store)
			export, err := svc.Export(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}

			if output != "" {
				if err := os.WriteFile(output, data, 0644); err != nil {
					return fmt.Errorf("write file: %w", err)
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported to %s\n", output)
				return nil
			}

			data = append(data, '\n')
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: stdout)")
	return cmd
}

func newTeamImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "import <file-or-url>",
		Short:       "Import a team and all sub-resources from a JSON file or URL",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := readSource(args[0])
			if err != nil {
				return err
			}

			var export domain.TeamExport
			if err := json.Unmarshal(data, &export); err != nil {
				return fmt.Errorf("parse JSON: %w", err)
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			svc := team.New(store)
			imported, err := svc.Import(cmd.Context(), export)
			if err != nil {
				return err
			}

			updated, err := svc.BuildPlan(cmd.Context(), imported.ID)
			if err != nil {
				return err
			}

			return printJSON(updated)
		},
	}
}

// readSource reads JSON bytes from a local file or an HTTP(S) URL.
func readSource(src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		resp, err := http.Get(src)
		if err != nil {
			return nil, fmt.Errorf("fetch URL: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch URL: %s", resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	}
	return os.ReadFile(src)
}
