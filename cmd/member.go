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
	rootCmd.AddCommand(newMemberCmd())
}

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage members",
	}
	cmd.AddCommand(newMemberCreateCmd())
	cmd.AddCommand(newMemberListCmd())
	cmd.AddCommand(newMemberUpdateCmd())
	cmd.AddCommand(newMemberDeleteCmd())
	cmd.AddCommand(newMemberWorkspaceCmd())
	cmd.AddCommand(newMemberRunCmd())
	return cmd
}

func newMemberCreateCmd() *cobra.Command {
	var name, command, claudeMd, claudeSettings, repo string
	var skills []string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a member",

		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			body := map[string]any{
				"name":    name,
				"command": command,
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
	cmd.Flags().StringVar(&command, "command", "", "Command (binary + CLI flags, e.g. \"claude --dangerously-skip-permissions\")")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "Claude md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skill IDs (comma-separated)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func newMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all members",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			members, err := client.ListMembers(owner)
			if err != nil {
				return err
			}
			return printJSON(members)
		},
	}
}

func newMemberUpdateCmd() *cobra.Command {
	var name, command, claudeMd, claudeSettings, repo string
	var skills []string

	cmd := &cobra.Command{
		Use:         "update <name>",
		Short:       "Update a member by name",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
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
	cmd.Flags().StringVar(&command, "command", "", "New command (binary + CLI flags)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New claude md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "New skill IDs (comma-separated)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL")
	return cmd
}

func newMemberDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <name>",
		Short:       "Delete a member by name",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			if err := client.DeleteMember(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newMemberWorkspaceCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "workspace <member-name>",
		Short: "Create workspace for a member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()
			writer := appws.NewWriter(client, owner)

			base := dir
			if base == "" {
				base = "."
			}

			if err := writer.PrepareMember(base, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "prepared",
				"member": args[0],
				"dir":    base,
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Base directory for workspace (default: current directory)")
	return cmd
}

func newMemberRunCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "run <member-name>",
		Short: "Create workspace and run a single member",
		Long: `Create workspace (idempotent) and run a single member.
This prepares the workspace files and launches the agent in a tmux session.`,
		Args:        cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			memberID := args[0]
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

			// 1. Workspace (idempotent) -- skip if project dir already exists
			projectDir := filepath.Join(absBase, "project")
			if _, err := os.Stat(projectDir); os.IsNotExist(err) {
				writer := appws.NewWriter(client, owner)
				if err := writer.PrepareMember(absBase, memberID); err != nil {
					return fmt.Errorf("prepare workspace: %w", err)
				}
			}

			// 2. Get member spec for command
			member, err := client.GetMember(owner, memberID)
			if err != nil {
				return fmt.Errorf("get member: %w", err)
			}

			// 3. Create Run on server
			runResp, err := client.CreateRun(map[string]any{
				"member_id": member.ID,
			})
			if err != nil {
				return fmt.Errorf("create run: %w", err)
			}
			runID := runResp.ID
			runIDStr := strconv.FormatInt(runID, 10)
			runName := apprun.SessionName(member.Name, runIDStr)

			// 4. Build env vars + command
			runPlanPath := filepath.Join(absBase, ".clier", runIDStr+".json")
			envVars := buildMemberEnv(runID, member.ID, member.Name, runPlanPath, absBase)
			projectPath := filepath.Join(absBase, "project")
			fullCommand := buildFullCommand(envVars, member.Command, projectPath)

			// 5. Build RunPlan
			plan := &apprun.RunPlan{
				Session: runName,
				Members: []apprun.MemberTerminal{{
					Name:    member.Name,
					Window:  0,
					Cwd:     projectPath,
					Command: fullCommand,
				}},
			}

			// 6. Save .clier/{RUN_ID}.json
			if err := apprun.SavePlan(absBase, runIDStr, plan); err != nil {
				return fmt.Errorf("save plan: %w", err)
			}

			// 7. Launch tmux
			term := terminal.NewTmuxTerminal(terminal.NewLocalRefStore(""))
			domainPlans := []domain.MemberPlan{{
				TeamMemberID: member.ID,
				MemberName:   member.Name,
				Terminal:     domain.TerminalPlan{Command: fullCommand},
				Workspace:    domain.WorkspacePlan{Memberspace: absBase},
			}}
			if err := term.Launch(runIDStr, plan.Session, domainPlans); err != nil {
				return fmt.Errorf("launch: %w", err)
			}

			return printJSON(map[string]any{
				"run_id":  runID,
				"session": plan.Session,
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Base directory for workspace (default: current directory)")
	return cmd
}
