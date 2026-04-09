package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
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
		Use:     "member",
		Short:   "Manage members",
		GroupID: rootGroupServer,
		Long: `Manage member resources stored on clier-server.

Server-backed subcommands:
  list, view, create, edit, delete, fork

Local runtime subcommands:
  clone, run

Use the local runtime subcommands after selecting a member to
materialize files into the current directory and launch a local run.

` + "`member clone`" + ` is one-way: it writes a local runnable worktree,
but does not sync local file edits back to clier-server. Update server
resources with explicit resource commands, then remove and re-clone when
you want a fresh local copy.`,
	}
	cmd.AddGroup(
		&cobra.Group{ID: subGroupServer, Title: "Server-Backed Member Commands"},
		&cobra.Group{ID: subGroupRuntime, Title: "Local Runtime Commands"},
	)
	cmd.AddCommand(newMemberListCmd())
	cmd.AddCommand(newMemberViewCmd())
	cmd.AddCommand(newMemberCreateCmd())
	cmd.AddCommand(newMemberEditCmd())
	cmd.AddCommand(newMemberDeleteCmd())
	cmd.AddCommand(newMemberForkCmd())
	cmd.AddCommand(newMemberCloneCmd())
	cmd.AddCommand(newMemberRunCmd())
	return cmd
}

func newMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [owner]",
		Short:   "List members from clier-server",
		Long:    "List your members, or another user's members if [owner] is given.",
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
		Use:     "view <[owner/]name>",
		Short:   "View a member from clier-server",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
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
		Use:     "create",
		Short:   "Create a member on clier-server",
		GroupID: subGroupServer,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			claudeMdID, err := parseOptionalInt64(claudeMd)
			if err != nil {
				return fmt.Errorf("parse --claude-md: %w", err)
			}
			claudeSettingsID, err := parseOptionalInt64(claudeSettings)
			if err != nil {
				return fmt.Errorf("parse --claude-settings: %w", err)
			}
			skillIDs := make([]int64, 0, len(skills))
			for _, raw := range skills {
				id, err := parseOptionalInt64(raw)
				if err != nil {
					return fmt.Errorf("parse --skills %q: %w", raw, err)
				}
				if id == nil {
					return fmt.Errorf("parse --skills %q: value must not be empty", raw)
				}
				skillIDs = append(skillIDs, *id)
			}
			body := api.MemberMutationRequest{
				Name:             name,
				AgentType:        agentType,
				Command:          command,
				GitRepoURL:       repo,
				ClaudeMdID:       claudeMdID,
				ClaudeSettingsID: claudeSettingsID,
				SkillIDs:         skillIDs,
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
	_ = cmd.MarkFlagRequired("agent-type")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func newMemberEditCmd() *cobra.Command {
	var name, agentType, command, claudeMd, claudeSettings, repo string
	var skills []string

	cmd := &cobra.Command{
		Use:     "edit <name>",
		Short:   "Edit a member on clier-server",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			current, err := client.GetMember(owner, args[0])
			if err != nil {
				return err
			}
			body := api.MemberMutationRequest{
				Name:             current.Name,
				AgentType:        current.AgentType,
				Command:          current.Command,
				GitRepoURL:       current.GitRepoURL,
				ClaudeMdID:       current.ClaudeMdID,
				ClaudeSettingsID: current.ClaudeSettingsID,
				SkillIDs:         resourceIDs(current.Skills),
			}
			if cmd.Flags().Changed("name") {
				body.Name = name
			}
			if cmd.Flags().Changed("agent-type") {
				body.AgentType = agentType
			}
			if cmd.Flags().Changed("command") {
				body.Command = command
			}
			if cmd.Flags().Changed("claude-md") {
				body.ClaudeMdID, err = parseOptionalInt64(claudeMd)
				if err != nil {
					return fmt.Errorf("parse --claude-md: %w", err)
				}
			}
			if cmd.Flags().Changed("skills") {
				body.SkillIDs = make([]int64, 0, len(skills))
				for _, raw := range skills {
					id, err := parseOptionalInt64(raw)
					if err != nil {
						return fmt.Errorf("parse --skills %q: %w", raw, err)
					}
					if id == nil {
						return fmt.Errorf("parse --skills %q: value must not be empty", raw)
					}
					body.SkillIDs = append(body.SkillIDs, *id)
				}
			}
			if cmd.Flags().Changed("claude-settings") {
				body.ClaudeSettingsID, err = parseOptionalInt64(claudeSettings)
				if err != nil {
					return fmt.Errorf("parse --claude-settings: %w", err)
				}
			}
			if cmd.Flags().Changed("repo") {
				body.GitRepoURL = repo
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
		Use:     "delete <name>",
		Short:   "Delete a member from clier-server",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
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
		Use:     "fork <owner/name>",
		Short:   "Fork a member on clier-server to your namespace",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
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

func newMemberCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "clone <[owner/]name>",
		Aliases: []string{"workspace"},
		Short:   "Create a local member clone under ./<owner>/<name>",
		GroupID: subGroupRuntime,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			writer := appws.NewWriter(client, owner)
			base, err := resolveCloneCreateBase(cloneTarget{
				Kind:  resourceKindMember,
				Owner: owner,
				Name:  name,
			})
			if err != nil {
				return err
			}

			if err := writer.PrepareMember(base, name); err != nil {
				return err
			}
			meta, err := buildMemberCloneMetadata(client, owner, name)
			if err != nil {
				return err
			}
			if err := appws.SaveCloneMetadata(base, meta); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "cloned",
				"member": name,
				"dir":    base,
			})
		},
	}
}

func newMemberRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "run",
		Short:   "Launch a local member run from the current clone root",
		GroupID: subGroupRuntime,
		Long: `Launch a single member from the current clone root.
This command is local runtime, not a clier-server run API call.

The current directory must be the member clone root that directly owns
` + "`.clier/clone.json`" + `. Run ` + "`member clone`" + `
first, then ` + "`cd`" + ` into that clone root before starting a run.

The clone is a one-way local worktree. To refresh it from server
resources, remove the clone and create it again.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()

			absBase, meta, err := requireCurrentCloneRootKind(resourceKindMember, "`clier member run`")
			if err != nil {
				return err
			}
			member, err := client.GetMember(meta.Owner, meta.Name)
			if err != nil {
				return fmt.Errorf("get member: %w", err)
			}
			repoPath := absBase
			prepared, err := appws.IsPreparedRoot(member.GitRepoURL, repoPath)
			if err != nil {
				return err
			}
			if !prepared {
				writer := appws.NewWriter(client, meta.Owner)
				if err := writer.PrepareMember(absBase, meta.Name); err != nil {
					return fmt.Errorf("prepare clone: %w", err)
				}
				meta, err := buildMemberCloneMetadata(client, meta.Owner, meta.Name)
				if err != nil {
					return err
				}
				if err := appws.SaveCloneMetadata(absBase, meta); err != nil {
					return err
				}
			}

			runID, err := newRunID()
			if err != nil {
				return err
			}
			runName := apprun.SessionName(member.Name, runID)

			envVars := buildMemberEnv(runID, member.ID, nil, member.Name)
			fullCommand := buildFullCommand(envVars, member.Command, repoPath)
			domainPlans := []domain.MemberPlan{{
				TeamMemberID: member.ID,
				MemberName:   member.Name,
				Terminal:     domain.TerminalPlan{Command: fullCommand},
				Workspace:    domain.WorkspacePlan{Memberspace: absBase},
			}}
			runner := apprun.NewRunner(newTerminal())
			plan, err := runner.Run(absBase, runID, runName, domainPlans)
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
