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
	cmd.AddCommand(newTeamListCmd())
	cmd.AddCommand(newTeamViewCmd())
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamEditCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	cmd.AddCommand(newTeamForkCmd())
	cmd.AddCommand(newTeamWorkspaceCmd())
	cmd.AddCommand(newTeamRunCmd())
	return cmd
}

func newTeamListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [owner]",
		Short: "List teams",
		Long:  "List your teams, or another user's teams if [owner] is given.",
		Args:  cobra.MaximumNArgs(1),
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
		Use:   "view <[owner/]name>",
		Short: "View a team",
		Args:  cobra.ExactArgs(1),
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
	var name, rootMemberID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
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

func newTeamEditCmd() *cobra.Command {
	var name, rootMemberID string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()

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
		Use:   "delete <name>",
		Short: "Delete a team",
		Args:  cobra.ExactArgs(1),
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

func newTeamForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <owner/name>",
		Short: "Fork a team to your namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			owner, name := parseOwnerName(args[0])
			resp, err := client.ForkTeam(owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}

func newTeamWorkspaceCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "workspace <[owner/]name>",
		Short: "Create workspaces for all team members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			writer := appws.NewWriter(client, owner)

			base := dir
			if base == "" {
				base = "."
			}

			if err := writer.PrepareTeam(base, name); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "prepared",
				"team":   name,
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
		Use:   "run <[owner/]name>",
		Short: "Create workspaces and run the team",
		Long: `Create workspaces (idempotent) for all team members and start a run.
Each member gets its own tmux window within a single session.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			_ = requireLogin()

			base := dir
			if base == "" {
				base = "."
			}
			absBase, err := filepath.Abs(base)
			if err != nil {
				return fmt.Errorf("resolve base path: %w", err)
			}

			team, err := client.GetTeam(owner, name)
			if err != nil {
				return fmt.Errorf("get team: %w", err)
			}

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

			runResp, err := client.CreateRun(map[string]any{
				"name":    team.Name,
				"team_id": team.ID,
			})
			if err != nil {
				return fmt.Errorf("create run: %w", err)
			}
			runID := runResp.ID
			runIDStr := strconv.FormatInt(runID, 10)
			runName := apprun.SessionName(team.Name, runIDStr)

			runPlanPath := filepath.Join(absBase, ".clier", runIDStr+".json")
			var memberTerminals []apprun.MemberTerminal
			var domainPlans []domain.MemberPlan

			for i, tm := range team.TeamMembers {
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

			if err := apprun.SavePlan(absBase, runIDStr, plan); err != nil {
				return fmt.Errorf("save plan: %w", err)
			}

			term := terminal.NewTmuxTerminal(newRefStore())
			if err := term.Launch(runIDStr, plan.Session, domainPlans); err != nil {
				return fmt.Errorf("launch: %w", err)
			}

			return printJSON(map[string]any{
				"run_id":  runID,
				"session": plan.Session,
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Base directory for workspaces (default: current directory)")
	return cmd
}
