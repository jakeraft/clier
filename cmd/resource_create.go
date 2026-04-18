package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCreateCmd())
}

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <kind>",
		Short:   "Create a new resource",
		Long:    `Create a new resource. Kind is required as a subcommand: team, skill, instruction, claude-settings, codex-settings.`,
		GroupID: rootGroupResources,
		RunE:    subcommandRequired,
	}
	cmd.AddCommand(newCreateTeamCmd())
	cmd.AddCommand(newCreateSkillCmd())
	cmd.AddCommand(newCreateInstructionCmd())
	cmd.AddCommand(newCreateClaudeSettingsCmd())
	cmd.AddCommand(newCreateCodexSettingsCmd())
	return cmd
}

func newCreateTeamCmd() *cobra.Command {
	var ownerFlag, name, command, instruction, claudeSettings, codexSettings, repo, summary string
	var skills, children []string

	cmd := &cobra.Command{
		Use:   "team",
		Short: "Create a new team",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			instructionRef, err := parseOptionalResourceRefRequest(instruction)
			if err != nil {
				return fmt.Errorf("parse --instruction: %w", err)
			}
			claudeSettingsRef, err := parseOptionalResourceRefRequest(claudeSettings)
			if err != nil {
				return fmt.Errorf("parse --claude-settings: %w", err)
			}
			codexSettingsRef, err := parseOptionalResourceRefRequest(codexSettings)
			if err != nil {
				return fmt.Errorf("parse --codex-settings: %w", err)
			}
			skillRefs, err := parseResourceRefSlice(skills)
			if err != nil {
				return fmt.Errorf("parse --skill: %w", err)
			}
			childRefs, err := parseChildRefSpecs(children)
			if err != nil {
				return err
			}
			body := remoteapi.TeamWriteRequest{
				Name:           name,
				Command:        command,
				GitRepoURL:     repo,
				Instruction:    instructionRef,
				ClaudeSettings: claudeSettingsRef,
				CodexSettings:  codexSettingsRef,
				Skills:         skillRefs,
				Children:       childRefs,
				Summary:        summary,
			}
			resp, err := svc.CreateResource(remoteapi.KindTeam, owner, body)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringVar(&command, "command", "", "Command (binary + CLI flags)")
	cmd.Flags().StringVar(&instruction, "instruction", "", "Instruction resource ref as <owner/name>@<version>")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "Skill ref as <owner/name>@<version>; repeat for each skill")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "Claude settings resource ref as <owner/name>@<version>")
	cmd.Flags().StringVar(&codexSettings, "codex-settings", "", "Codex settings resource ref as <owner/name>@<version>")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
	cmd.Flags().StringSliceVar(&children, "child", nil, "Child team ref as <owner/name>@<version>; repeat for each child")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newCreateSkillCmd() *cobra.Command {
	var ownerFlag, name, content, summary string

	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Create a new skill",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := svc.CreateResource(remoteapi.KindSkill, owner, remoteapi.ContentWriteRequest{
				Name:    name,
				Content: content,
				Summary: summary,
			})
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))
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

func newCreateInstructionCmd() *cobra.Command {
	var ownerFlag, name, content, summary string

	cmd := &cobra.Command{
		Use:   "instruction",
		Short: "Create a new instruction resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := svc.CreateResource(remoteapi.KindInstruction, owner, remoteapi.ContentWriteRequest{
				Name:    name,
				Content: content,
				Summary: summary,
			})
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))
		},
	}
	cmd.Flags().StringVar(&ownerFlag, "owner", "", "Resource owner (defaults to logged-in user)")
	cmd.Flags().StringVar(&name, "name", "", "Instruction name")
	cmd.Flags().StringVar(&content, "content", "", "Instruction content")
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
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := svc.CreateResource(remoteapi.KindClaudeSettings, owner, remoteapi.ContentWriteRequest{
				Name:    name,
				Content: content,
				Summary: summary,
			})
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))
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

func newCreateCodexSettingsCmd() *cobra.Command {
	var ownerFlag, name, content, summary string
	cmd := &cobra.Command{
		Use:   "codex-settings",
		Short: "Create a new Codex settings resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, err := resolveOwner(ownerFlag)
			if err != nil {
				return err
			}
			resp, err := svc.CreateResource(remoteapi.KindCodexSettings, owner, remoteapi.ContentWriteRequest{
				Name: name, Content: content, Summary: summary,
			})
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))
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
