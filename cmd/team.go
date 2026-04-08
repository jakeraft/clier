package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	apprun "github.com/jakeraft/clier/internal/app/run"
	appws "github.com/jakeraft/clier/internal/app/workspace"
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
	cmd.AddCommand(newTeamWorkspaceCmd())
	cmd.AddCommand(newTeamRunCmd())
	return cmd
}

func newTeamCreateCmd() *cobra.Command {
	var name, rootMemberID string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a team",

		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			resp, err := client.CreateTeam(owner, map[string]string{
				"name":           name,
				"root_member_id": rootMemberID,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
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
			client := newAPIClient()
			owner := resolveOwner()

			teams, err := client.ListTeams(owner)
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
		Use:         "update <name>",
		Short:       "Update a team by name",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			body := map[string]string{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("root-member") {
				body["root_team_member_id"] = rootMemberID
			}

			resp, err := client.UpdateTeam(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New team name")
	cmd.Flags().StringVar(&rootMemberID, "root-member", "", "New root member ID")
	return cmd
}

func newTeamDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <name>",
		Short:       "Delete a team by name",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			if err := client.DeleteTeam(owner, args[0]); err != nil {
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
	cmd.AddCommand(newTeamMemberListCmd())
	return cmd
}

func newTeamMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <team-name>",
		Short: "List members of a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			team, err := client.GetTeam(owner, args[0])
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
	cmd.AddCommand(newTeamRelationListCmd())
	return cmd
}

func newTeamRelationListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <team-name>",
		Short: "List relations of a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			team, err := client.GetTeam(owner, args[0])
			if err != nil {
				return err
			}
			return printJSON(team.Relations)
		},
	}
}

func newTeamWorkspaceCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "workspace <team-name>",
		Short: "Create workspaces for all team members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()
			writer := appws.NewWriter(client, owner)

			base := dir
			if base == "" {
				base = "."
			}

			if err := writer.PrepareTeam(base, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "prepared",
				"team":   args[0],
				"dir":    base,
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Base directory for workspaces (default: current directory)")
	return cmd
}

func newTeamRunCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "run <team-name>",
		Short: "Create workspaces and run the team",
		Long: `Create workspaces (idempotent) for all team members and start a run.
Each member gets its own tmux window within a single session.`,
		Args:        cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]
			client := newAPIClient()
			owner := resolveOwner()

			base := dir
			if base == "" {
				base = "."
			}
			absBase, err := filepath.Abs(base)
			if err != nil {
				return fmt.Errorf("resolve base path: %w", err)
			}

			// 1. Get team definition
			team, err := client.GetTeam(owner, teamID)
			if err != nil {
				return fmt.Errorf("get team: %w", err)
			}

			// 2. Workspace (idempotent) -- skip members whose project dir already exists
			writer := appws.NewWriter(client, owner)
			for _, tm := range team.TeamMembers {
				memberBase := filepath.Join(absBase, tm.Name)
				projectDir := filepath.Join(memberBase, "project")
				if _, statErr := os.Stat(projectDir); os.IsNotExist(statErr) {
					if err := writer.PrepareMember(memberBase, tm.Member.Name); err != nil {
						return fmt.Errorf("prepare member %s: %w", tm.Name, err)
					}
				}
			}

			// 3. Create Run on server
			runResp, err := client.CreateRun(map[string]any{
				"team_id": team.ID,
			})
			if err != nil {
				return fmt.Errorf("create run: %w", err)
			}
			runID := strconv.FormatInt(runResp.ID, 10)
			runName := apprun.SessionName(team.Name, runID)

			// 4. Build RunPlan + domain plans
			runPlanPath := filepath.Join(absBase, ".clier", runID+".json")
			var memberTerminals []apprun.MemberTerminal
			var domainPlans []domain.MemberPlan

			for i, tm := range team.TeamMembers {
				// Get member spec for command
				member, err := client.GetMember(tm.Member.Owner, tm.Member.Name)
				if err != nil {
					return fmt.Errorf("get member %s: %w", tm.Name, err)
				}

				memberBase := filepath.Join(absBase, tm.Name)
				projectPath := filepath.Join(memberBase, "project")

				envVars := buildMemberEnv(runID, tm.ID, tm.Name, runPlanPath, memberBase)
				fullCommand := buildFullCommand(envVars, member.Command, projectPath)

				memberTerminals = append(memberTerminals, apprun.MemberTerminal{
					Name:    tm.Name,
					Window:  i,
					Cwd:     projectPath,
					Command: fullCommand,
				})

				domainPlans = append(domainPlans, domain.MemberPlan{
					TeamMemberID: tm.ID,
					MemberName:   tm.Name,
					Terminal:     domain.TerminalPlan{Command: fullCommand},
					Workspace:    domain.WorkspacePlan{Memberspace: memberBase},
				})
			}

			plan := &apprun.RunPlan{
				Session: runName,
				Members: memberTerminals,
			}

			// 5. Save .clier/{RUN_ID}.json
			if err := apprun.SavePlan(absBase, runID, plan); err != nil {
				return fmt.Errorf("save plan: %w", err)
			}

			// 6. Launch tmux
			term := terminal.NewTmuxTerminal(terminal.NewLocalRefStore(""))
			if err := term.Launch(runID, plan.Session, domainPlans); err != nil {
				return fmt.Errorf("launch: %w", err)
			}

			return printJSON(map[string]string{
				"run_id":  runID,
				"session": plan.Session,
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Base directory for workspaces (default: current directory)")
	return cmd
}
