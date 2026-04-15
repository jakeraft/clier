package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	rootCmd.AddCommand(newEditCmd())
}

var kindAllowedFlags = map[api.ResourceKind]map[string]bool{
	api.KindMember:         {"name": true, "summary": true, "command": true, "repo": true, "claude-md": true, "claude-settings": true, "codex-md": true, "codex-settings": true, "skill": true},
	api.KindTeam:           {"name": true, "summary": true, "member": true, "relation": true},
	api.KindSkill:          {"name": true, "summary": true, "content": true},
	api.KindClaudeMd:       {"name": true, "summary": true, "content": true},
	api.KindClaudeSettings: {"name": true, "summary": true, "content": true},
	api.KindCodexMd:        {"name": true, "summary": true, "content": true},
	api.KindCodexSettings:  {"name": true, "summary": true, "content": true},
}

func validateEditFlags(cmd *cobra.Command, kind api.ResourceKind) error {
	allowed := kindAllowedFlags[kind]
	var invalid []string
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if !allowed[f.Name] {
			invalid = append(invalid, "--"+f.Name)
		}
	})
	if len(invalid) > 0 {
		return fmt.Errorf("flags %v not applicable to resource kind %q", invalid, kind)
	}
	return nil
}

func newEditCmd() *cobra.Command {
	var name, command, content, claudeMd, claudeSettings, codexMd, codexSettings, repo, summary string
	var skills []string
	var teamMembers, relations []string

	cmd := &cobra.Command{
		Use:   "edit <[owner/]name>",
		Short: "Update a resource (auto-detects kind)",
		Long: `Update a resource. The resource kind is detected automatically
via a GET request, and only the flags you provide are sent as changes.
Owner defaults to the logged-in user when not specified.`,
		GroupID: rootGroupResources,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, err := parseOwnerName(args[0])
			if err != nil {
				return err
			}

			// Detect kind via GET.
			res, err := client.GetResource(owner, name)
			if err != nil {
				return fmt.Errorf("look up resource %q: %w", args[0], err)
			}
			kind := api.ResourceKind(res.Kind)

			// Validate that only kind-appropriate flags are used.
			if err := validateEditFlags(cmd, kind); err != nil {
				return err
			}

			switch kind {
			case api.KindMember:
				body := api.MemberPatchRequest{}
				if cmd.Flags().Changed("name") {
					body.Name = &name
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
				if cmd.Flags().Changed("codex-md") {
					ref, err := parseOptionalResourceRefRequest(codexMd)
					if err != nil {
						return fmt.Errorf("parse --codex-md: %w", err)
					}
					body.CodexMd = ref
				}
				if cmd.Flags().Changed("codex-settings") {
					ref, err := parseOptionalResourceRefRequest(codexSettings)
					if err != nil {
						return fmt.Errorf("parse --codex-settings: %w", err)
					}
					body.CodexSettings = ref
				}
				if cmd.Flags().Changed("skill") {
					body.Skills = make([]api.ResourceRefRequest, 0, len(skills))
					for _, raw := range skills {
						ref, err := parseOptionalResourceRefRequest(raw)
						if err != nil {
							return fmt.Errorf("parse --skill %q: %w", raw, err)
						}
						if ref == nil {
							return fmt.Errorf("parse --skill %q: value must not be empty", raw)
						}
						body.Skills = append(body.Skills, *ref)
					}
				}
				resp, err := client.PatchResource(api.KindMember, owner, name, &body)
				if err != nil {
					return err
				}
				return printJSON(resp)

			case api.KindTeam:
				body := api.TeamPatchRequest{}
				if cmd.Flags().Changed("name") {
					body.Name = &name
				}
				if cmd.Flags().Changed("summary") {
					body.Summary = &summary
				}
				if cmd.Flags().Changed("member") {
					members, err := parseTeamMemberSpecs(teamMembers)
					if err != nil {
						return err
					}
					body.TeamMembers = members
					if !cmd.Flags().Changed("relation") {
						body.Relations = []api.TeamRelationRequest{}
					}
				}
				if cmd.Flags().Changed("relation") {
					parsed, err := parseTeamRelationSpecs(relations)
					if err != nil {
						return err
					}
					body.Relations = parsed
				}
				resp, err := client.PatchResource(api.KindTeam, owner, name, &body)
				if err != nil {
					return err
				}
				return printJSON(resp)

			case api.KindSkill, api.KindClaudeMd, api.KindClaudeSettings, api.KindCodexMd, api.KindCodexSettings:
				body := api.ContentPatchRequest{}
				if cmd.Flags().Changed("name") {
					body.Name = &name
				}
				if cmd.Flags().Changed("content") {
					body.Content = &content
				}
				if cmd.Flags().Changed("summary") {
					body.Summary = &summary
				}
				resp, err := client.PatchResource(kind, owner, name, &body)
				if err != nil {
					return err
				}
				return printJSON(resp)

			default:
				return fmt.Errorf("unsupported resource kind %q", res.Kind)
			}
		},
	}
	// Superset of all kind-specific flags.
	cmd.Flags().StringVar(&name, "name", "", "New resource name")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	cmd.Flags().StringVar(&content, "content", "", "New content (skill, claude-md, claude-settings, codex-md, codex-settings)")
	cmd.Flags().StringVar(&command, "command", "", "New command (member)")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL (member)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New claude md ref as <id>@<version> (member)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New claude settings ref as <id>@<version> (member)")
	cmd.Flags().StringVar(&codexMd, "codex-md", "", "New codex instruction ref as <id>@<version> (member)")
	cmd.Flags().StringVar(&codexSettings, "codex-settings", "", "New codex settings ref as <id>@<version> (member)")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "New skill ref as <id>@<version>; repeat for each (member)")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Replace team members with <member-id>@<version>; repeat for each (team)")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Replace relations with <from-member-id>:<to-member-id>; repeat for each (team)")
	return cmd
}
