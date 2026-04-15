package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCreateCmd())
}

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <kind>",
		Short:   "Create a new resource",
		Long:    `Create a new resource. Kind is required as a subcommand: member, team, skill, claude-md, claude-settings, codex-md, codex-settings.`,
		GroupID: rootGroupResources,
		RunE:    subcommandRequired,
	}
	cmd.AddCommand(newCreateMemberCmd())
	cmd.AddCommand(newCreateTeamCmd())
	cmd.AddCommand(newCreateSkillCmd())
	cmd.AddCommand(newCreateClaudeMdCmd())
	cmd.AddCommand(newCreateClaudeSettingsCmd())
	cmd.AddCommand(newCreateCodexMdCmd())
	cmd.AddCommand(newCreateCodexSettingsCmd())
	return cmd
}

func newCreateMemberCmd() *cobra.Command {
	var ownerFlag, name, command, claudeMd, claudeSettings, codexMd, codexSettings, repo, summary string
	var skills []string

	cmd := &cobra.Command{
		Use:   "member",
		Short: "Create a new member",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			claudeMdRef, err := parseOptionalResourceRefRequest(claudeMd)
			if err != nil {
				return fmt.Errorf("parse --claude-md: %w", err)
			}
			claudeSettingsRef, err := parseOptionalResourceRefRequest(claudeSettings)
			if err != nil {
				return fmt.Errorf("parse --claude-settings: %w", err)
			}
			codexMdRef, err := parseOptionalResourceRefRequest(codexMd)
			if err != nil {
				return fmt.Errorf("parse --codex-md: %w", err)
			}
			codexSettingsRef, err := parseOptionalResourceRefRequest(codexSettings)
			if err != nil {
				return fmt.Errorf("parse --codex-settings: %w", err)
			}
			skillRefs := make([]api.ResourceRefRequest, 0, len(skills))
			for _, raw := range skills {
				ref, err := parseOptionalResourceRefRequest(raw)
				if err != nil {
					return fmt.Errorf("parse --skill %q: %w", raw, err)
				}
				if ref == nil {
					return fmt.Errorf("parse --skill %q: value must not be empty", raw)
				}
				skillRefs = append(skillRefs, *ref)
			}
			body := api.MemberWriteRequest{
				Name:           name,
				Command:        command,
				GitRepoURL:     repo,
				ClaudeMd:       claudeMdRef,
				ClaudeSettings: claudeSettingsRef,
				CodexMd:        codexMdRef,
				CodexSettings:  codexSettingsRef,
				Skills:         skillRefs,
				Summary:        summary,
			}
			resp, err := client.CreateResource(api.KindMember, owner, body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Member name")
	cmd.Flags().StringVar(&command, "command", "", "Command (binary + CLI flags)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "Claude md resource ref as <owner/name>@<version>")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "Skill ref as <owner/name>@<version>; repeat for each skill")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "Claude settings resource ref as <owner/name>@<version>")
	cmd.Flags().StringVar(&codexMd, "codex-md", "", "Codex instruction resource ref as <owner/name>@<version>")
	cmd.Flags().StringVar(&codexSettings, "codex-settings", "", "Codex settings resource ref as <owner/name>@<version>")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func newCreateTeamCmd() *cobra.Command {
	var ownerFlag, name, summary string
	var teamMembers, relations []string

	cmd := &cobra.Command{
		Use:   "team",
		Short: "Create a new team",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			members, err := parseTeamMemberSpecs(teamMembers)
			if err != nil {
				return err
			}
			parsedRelations, err := parseTeamRelationSpecs(relations)
			if err != nil {
				return err
			}
			body := api.TeamWriteRequest{
				Name:        name,
				TeamMembers: members,
				Relations:   parsedRelations,
				Summary:     summary,
			}
			resp, err := client.CreateResource(api.KindTeam, owner, body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Team member as <owner/name>@<version>; repeat for each member")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Relation as <owner/from-name>:<owner/to-name>; repeat for each edge")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("member")
	return cmd
}

func newCreateSkillCmd() *cobra.Command {
	var ownerFlag, name, content, summary string

	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Create a new skill",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := client.CreateResource(api.KindSkill, owner, api.ContentWriteRequest{
				Name:    name,
				Content: content,
				Summary: summary,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Skill name")
	cmd.Flags().StringVar(&content, "content", "", "Skill content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newCreateClaudeMdCmd() *cobra.Command {
	var ownerFlag, name, content, summary string

	cmd := &cobra.Command{
		Use:   "claude-md",
		Short: "Create a new CLAUDE.md resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := client.CreateResource(api.KindClaudeMd, owner, api.ContentWriteRequest{
				Name:    name,
				Content: content,
				Summary: summary,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Claude md name")
	cmd.Flags().StringVar(&content, "content", "", "Claude md content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newCreateClaudeSettingsCmd() *cobra.Command {
	var ownerFlag, name, content, summary string

	cmd := &cobra.Command{
		Use:   "claude-settings",
		Short: "Create a new Claude settings resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := client.CreateResource(api.KindClaudeSettings, owner, api.ContentWriteRequest{
				Name:    name,
				Content: content,
				Summary: summary,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Settings name")
	cmd.Flags().StringVar(&content, "content", "", "Settings JSON content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newCreateCodexMdCmd() *cobra.Command {
	var ownerFlag, name, content, summary string
	cmd := &cobra.Command{
		Use:   "codex-md",
		Short: "Create a new Codex instruction resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := client.CreateResource(api.KindCodexMd, owner, api.ContentWriteRequest{
				Name: name, Content: content, Summary: summary,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Codex instruction name")
	cmd.Flags().StringVar(&content, "content", "", "Codex instruction content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newCreateCodexSettingsCmd() *cobra.Command {
	var ownerFlag, name, content, summary string
	cmd := &cobra.Command{
		Use:   "codex-settings",
		Short: "Create a new Codex settings resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := client.CreateResource(api.KindCodexSettings, owner, api.ContentWriteRequest{
				Name: name, Content: content, Summary: summary,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Settings name")
	cmd.Flags().StringVar(&content, "content", "", "Settings TOML content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}
