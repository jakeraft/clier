package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	apprun "github.com/jakeraft/clier/internal/app/run"
	appws "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newMemberCmd())
}

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage members",
	}
	cmd.AddCommand(newMemberListCmd())
	cmd.AddCommand(newMemberViewCmd())
	cmd.AddCommand(newMemberCreateCmd())
	cmd.AddCommand(newMemberEditCmd())
	cmd.AddCommand(newMemberDeleteCmd())
	cmd.AddCommand(newMemberForkCmd())
	cmd.AddCommand(newMemberWorkspaceCmd())
	cmd.AddCommand(newMemberRunCmd())
	return cmd
}

func newMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [owner]",
		Short: "List members",
		Long:  "List your members, or another user's members if [owner] is given.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			var owner string
			if len(args) == 1 {
				owner = args[0]
			} else {
				owner = requireLogin()
			}
			members, err := client.ListMembers(owner)
			if err != nil {
				return err
			}
			return printJSON(members)
		},
	}
}

func newMemberViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <[owner/]name>",
		Short: "View a member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			member, err := client.GetMember(owner, name)
			if err != nil {
				return err
			}
			return printJSON(member)
		},
	}
}

func newMemberCreateCmd() *cobra.Command {
	var name, agentType, command, claudeMd, claudeSettings, repo string
	var skills []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a member",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()

			body := map[string]any{
				"name":    name,
				"command": command,
			}
			if agentType != "" {
				body["agent_type"] = agentType
			}
			if claudeMd != "" {
				body["claude_md_id"] = claudeMd
			}
			if skills != nil {
				body["skill_ids"] = skills
			}
			if claudeSettings != "" {
				body["claude_settings_id"] = claudeSettings
			}
			if repo != "" {
				body["git_repo_url"] = repo
			}

			resp, err := client.CreateMember(owner, body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Member name")
	cmd.Flags().StringVar(&agentType, "agent-type", "", "Agent type (e.g. claude, codex)")
	cmd.Flags().StringVar(&command, "command", "", "Command (binary + CLI flags)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "Claude md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skill IDs (comma-separated)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func newMemberEditCmd() *cobra.Command {
	var name, agentType, command, claudeMd, claudeSettings, repo string
	var skills []string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()

			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("agent-type") {
				body["agent_type"] = agentType
			}
			if cmd.Flags().Changed("command") {
				body["command"] = command
			}
			if cmd.Flags().Changed("claude-md") {
				body["claude_md_id"] = claudeMd
			}
			if cmd.Flags().Changed("skills") {
				body["skill_ids"] = skills
			}
			if cmd.Flags().Changed("claude-settings") {
				body["claude_settings_id"] = claudeSettings
			}
			if cmd.Flags().Changed("repo") {
				body["git_repo_url"] = repo
			}

			resp, err := client.UpdateMember(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New member name")
	cmd.Flags().StringVar(&agentType, "agent-type", "", "New agent type")
	cmd.Flags().StringVar(&command, "command", "", "New command")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New claude md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "New skill IDs")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL")
	return cmd
}

func newMemberDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteMember(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newMemberForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <owner/name>",
		Short: "Fork a member to your namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			owner, name := parseOwnerName(args[0])
			resp, err := client.ForkMember(owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}

func newMemberWorkspaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workspace <[owner/]name>",
		Short: "Create workspace for a member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			writer := appws.NewWriter(client, owner)
			base, err := resolveWorkspaceBase()
			if err != nil {
				return err
			}

			if err := writer.PrepareMember(base, name); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "prepared",
				"member": name,
				"dir":    base,
			})
		},
	}
}

func newMemberRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <[owner/]name>",
		Short: "Create workspace and run a single member",
		Long: `Create workspace (idempotent) and run a single member.
This prepares the workspace files and launches the agent in a tmux session.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			_ = requireLogin()

			absBase, err := resolveWorkspaceBase()
			if err != nil {
				return err
			}

			projectDir := filepath.Join(absBase, "project")
			if _, err := os.Stat(projectDir); os.IsNotExist(err) {
				writer := appws.NewWriter(client, owner)
				if err := writer.PrepareMember(absBase, name); err != nil {
					return fmt.Errorf("prepare workspace: %w", err)
				}
			}

			member, err := client.GetMember(owner, name)
			if err != nil {
				return fmt.Errorf("get member: %w", err)
			}

			runResp, err := client.CreateRun(map[string]any{
				"name":      member.Name,
				"member_id": member.ID,
			})
			if err != nil {
				return fmt.Errorf("create run: %w", err)
			}
			runID := runResp.ID
			runIDStr := strconv.FormatInt(runID, 10)
			runName := apprun.SessionName(member.Name, runIDStr)

			runPlanPath := apprun.PlanPath(absBase, runIDStr)
			envVars := buildMemberEnv(runID, member.ID, member.Name, runPlanPath, absBase)
			projectPath := filepath.Join(absBase, "project")
			fullCommand := buildFullCommand(envVars, member.Command, projectPath)
			domainPlans := []domain.MemberPlan{{
				TeamMemberID: member.ID,
				MemberName:   member.Name,
				Terminal:     domain.TerminalPlan{Command: fullCommand},
				Workspace:    domain.WorkspacePlan{Memberspace: absBase},
			}}
			runner := apprun.NewRunner(newTerminal())
			plan, err := runner.Run(absBase, runIDStr, runName, domainPlans)
			if err != nil {
				return err
			}

			return printJSON(map[string]any{
				"run_id":  runID,
				"session": plan.Session,
			})
		},
	}
}
