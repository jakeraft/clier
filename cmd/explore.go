package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

type exploreResourceSpec struct {
	name          string
	kind          string
	short         string
	listShort     string
	versionsShort string
}

func init() {
	rootCmd.AddCommand(newExploreCmd())
}

func newExploreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "explore",
		Short:   "Browse and inspect resources",
		GroupID: rootGroupDiscovery,
		Long:    `Browse public resources or inspect a specific owner's resources.`,
	}

	for _, spec := range exploreResourceSpecs() {
		cmd.AddCommand(newExploreResourceCmd(spec))
	}
	return cmd
}

func exploreResourceSpecs() []exploreResourceSpec {
	return []exploreResourceSpec{
		{
			name:          "team",
			kind:          string(api.KindTeam),
			short:         "Show a team definition",
			listShort:     "List public teams or one owner's teams",
			versionsShort: "List a team's versions",
		},
		{
			name:          "member",
			kind:          string(api.KindMember),
			short:         "Show a member definition",
			listShort:     "List public members or one owner's members",
			versionsShort: "List a member's versions",
		},
		{
			name:          "skill",
			kind:          string(api.KindSkill),
			short:         "Show a skill definition",
			listShort:     "List public skills or one owner's skills",
			versionsShort: "List a skill's versions",
		},
		{
			name:          "claude-md",
			kind:          string(api.KindClaudeMd),
			short:         "Show a CLAUDE.md definition",
			listShort:     "List public CLAUDE.md files or one owner's files",
			versionsShort: "List a CLAUDE.md file's versions",
		},
		{
			name:          "claude-settings",
			kind:          string(api.KindClaudeSettings),
			short:         "Show a Claude settings definition",
			listShort:     "List public Claude settings or one owner's settings",
			versionsShort: "List a Claude settings file's versions",
		},
	}
}

func newExploreResourceCmd(spec exploreResourceSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.name + " <owner/name>",
		Short: spec.short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, err := parseExplicitOwnerName(args[0])
			if err != nil {
				return err
			}
			item, err := client.GetResource(owner, name)
			if err != nil {
				return err
			}
			return printJSON(item)
		},
	}
	cmd.AddCommand(newExploreResourceListCmd(spec))
	cmd.AddCommand(newExploreResourceVersionsCmd(spec))
	return cmd
}

func newExploreResourceListCmd(spec exploreResourceSpec) *cobra.Command {
	return &cobra.Command{
		Use:   "list [owner]",
		Short: spec.listShort,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			opts := api.ListOptions{Kind: spec.kind}
			if len(args) == 0 {
				items, err := client.ListPublicResources(opts)
				if err != nil {
					return err
				}
				return printJSON(items)
			}
			items, err := client.ListResources(args[0], opts)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newExploreResourceVersionsCmd(spec exploreResourceSpec) *cobra.Command {
	return &cobra.Command{
		Use:   "versions <owner/name>",
		Short: spec.versionsShort,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, err := parseExplicitOwnerName(args[0])
			if err != nil {
				return err
			}
			items, err := client.ListResourceVersions(owner, name)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}
