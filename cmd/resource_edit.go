package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newEditCmd())
}

func newEditCmd() *cobra.Command {
	var name, command, content, claudeMd, claudeSettings, repo, summary string
	var skills []string
	var teamMembers, relations []string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Update an owned resource (auto-detects kind)",
		Long: `Update a resource you own. The resource kind is detected automatically
via a GET request, and only the flags you provide are sent as changes.`,
		GroupID: rootGroupResources,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()

			// Detect kind via GET.
			res, err := client.GetResource(owner, args[0])
			if err != nil {
				return fmt.Errorf("look up resource %q: %w", args[0], err)
			}
			kind := api.ResourceKind(res.Kind)

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
				resp, err := client.PatchResource(api.KindMember, owner, args[0], &body)
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
				resp, err := client.PatchResource(api.KindTeam, owner, args[0], &body)
				if err != nil {
					return err
				}
				return printJSON(resp)

			case api.KindSkill, api.KindClaudeMd, api.KindClaudeSettings:
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
				resp, err := client.PatchResource(kind, owner, args[0], &body)
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
	cmd.Flags().StringVar(&content, "content", "", "New content (skill, claude-md, claude-settings)")
	cmd.Flags().StringVar(&command, "command", "", "New command (member)")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL (member)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New claude md ref as <id>@<version> (member)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New claude settings ref as <id>@<version> (member)")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "New skill ref as <id>@<version>; repeat for each (member)")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Replace team members with <member-id>@<version>; repeat for each (team)")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Replace relations with <from-member-id>:<to-member-id>; repeat for each (team)")
	return cmd
}
