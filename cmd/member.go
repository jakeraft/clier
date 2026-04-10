package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newMemberCmd())
}

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "member",
		Short:   "Manage agent definitions",
		GroupID: rootGroupServer,
		Long:    `Create, edit, and delete individual agent definitions on the server.`,
	}
	cmd.AddCommand(newMemberCreateCmd())
	cmd.AddCommand(newMemberEditCmd())
	cmd.AddCommand(newMemberDeleteCmd())
	return cmd
}

func newMemberCreateCmd() *cobra.Command {
	var name, agentType, command, claudeMd, claudeSettings, repo, summary string
	var skills []string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new member",

		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			claudeMdRef, err := parseOptionalResourceRefRequest(claudeMd)
			if err != nil {
				return fmt.Errorf("parse --claude-md: %w", err)
			}
			claudeSettingsRef, err := parseOptionalResourceRefRequest(claudeSettings)
			if err != nil {
				return fmt.Errorf("parse --claude-settings: %w", err)
			}
			skillRefs := make([]api.ResourceRefRequest, 0, len(skills))
			for _, raw := range skills {
				ref, err := parseOptionalResourceRefRequest(raw)
				if err != nil {
					return fmt.Errorf("parse --skills %q: %w", raw, err)
				}
				if ref == nil {
					return fmt.Errorf("parse --skills %q: value must not be empty", raw)
				}
				skillRefs = append(skillRefs, *ref)
			}
			body := api.MemberWriteRequest{
				Name:           name,
				AgentType:      agentType,
				Command:        command,
				GitRepoURL:     repo,
				ClaudeMd:       claudeMdRef,
				ClaudeSettings: claudeSettingsRef,
				Skills:         skillRefs,
				Summary:        summary,
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
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "Claude md resource ref as <id>@<version>")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skill refs as <id>@<version>")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "Claude settings resource ref as <id>@<version>")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("agent-type")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func newMemberEditCmd() *cobra.Command {
	var name, agentType, command, claudeMd, claudeSettings, repo, summary string
	var skills []string

	cmd := &cobra.Command{
		Use:     "edit <name>",
		Short:   "Update a member",

		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			body := api.MemberPatchRequest{}
			if cmd.Flags().Changed("name") {
				body.Name = &name
			}
			if cmd.Flags().Changed("agent-type") {
				body.AgentType = &agentType
			}
			if cmd.Flags().Changed("command") {
				body.Command = &command
			}
			if cmd.Flags().Changed("repo") {
				body.GitRepoURL = &repo
			}
			if cmd.Flags().Changed("summary") {
				body.Summary = &summary
			}
			if cmd.Flags().Changed("claude-md") {
				ref, err := parseOptionalResourceRefRequest(claudeMd)
				if err != nil {
					return fmt.Errorf("parse --claude-md: %w", err)
				}
				body.ClaudeMd = ref
			}
			if cmd.Flags().Changed("claude-settings") {
				ref, err := parseOptionalResourceRefRequest(claudeSettings)
				if err != nil {
					return fmt.Errorf("parse --claude-settings: %w", err)
				}
				body.ClaudeSettings = ref
			}
			if cmd.Flags().Changed("skills") {
				body.Skills = make([]api.ResourceRefRequest, 0, len(skills))
				for _, raw := range skills {
					ref, err := parseOptionalResourceRefRequest(raw)
					if err != nil {
						return fmt.Errorf("parse --skills %q: %w", raw, err)
					}
					if ref == nil {
						return fmt.Errorf("parse --skills %q: value must not be empty", raw)
					}
					body.Skills = append(body.Skills, *ref)
				}
			}

			resp, err := client.PatchMember(owner, args[0], &body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New member name")
	cmd.Flags().StringVar(&agentType, "agent-type", "", "New agent type")
	cmd.Flags().StringVar(&command, "command", "", "New command")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New claude md resource ref as <id>@<version>")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "New skill refs as <id>@<version>")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New Claude settings resource ref as <id>@<version>")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	return cmd
}

func newMemberDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a member",

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
