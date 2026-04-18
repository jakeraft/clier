package cmd

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	rootCmd.AddCommand(newEditCmd())
}

var kindAllowedFlags = map[remoteapi.ResourceKind]map[string]bool{
	remoteapi.KindTeam:           {"summary": true, "command": true, "repo": true, "instruction": true, "claude-settings": true, "codex-settings": true, "skill": true, "child": true},
	remoteapi.KindSkill:          {"summary": true, "content": true},
	remoteapi.KindInstruction:    {"summary": true, "content": true},
	remoteapi.KindClaudeSettings: {"summary": true, "content": true},
	remoteapi.KindCodexSettings:  {"summary": true, "content": true},
}

func validateEditFlags(cmd *cobra.Command, kind remoteapi.ResourceKind) error {
	allowed := kindAllowedFlags[kind]
	var invalid []string
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if !allowed[f.Name] {
			invalid = append(invalid, "--"+f.Name)
		}
	})
	if len(invalid) > 0 {
		return &domain.Fault{
			Kind: domain.KindInvalidArgument,
			Subject: map[string]string{
				"detail": "flags " + strings.Join(invalid, ", ") + " not applicable to resource kind " + string(kind),
			},
		}
	}
	return nil
}

func newEditCmd() *cobra.Command {
	var command, content, instruction, claudeSettings, codexSettings, repo, summary string
	var skills, children []string

	cmd := &cobra.Command{
		Use:   "edit <owner/name>",
		Short: "Update a resource (auto-detects kind)",
		Long: `Update a resource. The resource kind is detected automatically
via a GET request, and only the flags you provide are sent as changes.`,
		GroupID: rootGroupResources,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newRemoteCatalogService()
			if err != nil {
				return err
			}
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}

			// Detect kind via GET.
			res, err := svc.GetResource(owner, name)
			if err != nil {
				return fmt.Errorf("look up resource %q: %w", args[0], err)
			}
			kind := remoteapi.ResourceKind(res.Kind)

			// Validate that only kind-appropriate flags are used.
			if err := validateEditFlags(cmd, kind); err != nil {
				return err
			}

			switch kind {
			case remoteapi.KindTeam:
				body := remoteapi.TeamPatchRequest{}
				if cmd.Flags().Changed("command") {
					body.Command = &command
				}
				if cmd.Flags().Changed("repo") {
					body.GitRepoURL = &repo
				}
				if cmd.Flags().Changed("summary") {
					body.Summary = &summary
				}
				if cmd.Flags().Changed("instruction") {
					ref, err := parseOptionalResourceRefRequest(instruction)
					if err != nil {
						return fmt.Errorf("parse --instruction: %w", err)
					}
					body.Instruction = ref
				}
				if cmd.Flags().Changed("claude-settings") {
					ref, err := parseOptionalResourceRefRequest(claudeSettings)
					if err != nil {
						return fmt.Errorf("parse --claude-settings: %w", err)
					}
					body.ClaudeSettings = ref
				}
				if cmd.Flags().Changed("codex-settings") {
					ref, err := parseOptionalResourceRefRequest(codexSettings)
					if err != nil {
						return fmt.Errorf("parse --codex-settings: %w", err)
					}
					body.CodexSettings = ref
				}
				if cmd.Flags().Changed("skill") {
					refs, err := parseResourceRefSlice(skills)
					if err != nil {
						return fmt.Errorf("parse --skill: %w", err)
					}
					body.Skills = refs
				}
				if cmd.Flags().Changed("child") {
					parsed, err := parseChildRefSpecs(children)
					if err != nil {
						return err
					}
					body.Children = parsed
				}
				resp, err := svc.PatchResource(remoteapi.KindTeam, owner, name, &body)
				if err != nil {
					return err
				}
				return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))

			case remoteapi.KindSkill, remoteapi.KindInstruction, remoteapi.KindClaudeSettings, remoteapi.KindCodexSettings:
				body := remoteapi.ContentPatchRequest{}
				if cmd.Flags().Changed("content") {
					body.Content = &content
				}
				if cmd.Flags().Changed("summary") {
					body.Summary = &summary
				}
				resp, err := svc.PatchResource(kind, owner, name, &body)
				if err != nil {
					return err
				}
				return present.Success(cmd.OutOrStdout(), view.ResourceOf(resp))

			default:
				return &domain.Fault{
					Kind:    domain.KindUnsupportedKind,
					Subject: map[string]string{"resource_kind": res.Kind},
				}
			}
		},
	}
	// Superset of all kind-specific flags.
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	cmd.Flags().StringVar(&content, "content", "", "New content (skill, instruction, claude-settings, codex-settings)")
	cmd.Flags().StringVar(&command, "command", "", "New command (team)")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL (team)")
	cmd.Flags().StringVar(&instruction, "instruction", "", "New instruction ref as <owner/name>@<version> (team)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New claude settings ref as <owner/name>@<version> (team)")
	cmd.Flags().StringVar(&codexSettings, "codex-settings", "", "New codex settings ref as <owner/name>@<version> (team)")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "New skill ref as <owner/name>@<version>; repeat for each (team)")
	cmd.Flags().StringSliceVar(&children, "child", nil, "Child team ref as <owner/name>@<version>; repeat for each (team)")
	return cmd
}
