package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

type exploreResourceSpec struct {
	name          string
	short         string
	listShort     string
	versionsShort string
	get           func(*api.Client, string, string) (any, error)
	listPublic    func(*api.Client) (any, error)
	listByOwner   func(*api.Client, string) (any, error)
	listVersions  func(*api.Client, string, string) (any, error)
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
			short:         "Show a team definition",
			listShort:     "List public teams or one owner's teams",
			versionsShort: "List a team's versions",
			get: func(client *api.Client, owner, name string) (any, error) {
				return client.GetTeam(owner, name)
			},
			listPublic: func(client *api.Client) (any, error) {
				return client.ListPublicTeams()
			},
			listByOwner: func(client *api.Client, owner string) (any, error) {
				return client.ListTeams(owner)
			},
			listVersions: func(client *api.Client, owner, name string) (any, error) {
				return client.ListTeamVersions(owner, name)
			},
		},
		{
			name:          "member",
			short:         "Show a member definition",
			listShort:     "List public members or one owner's members",
			versionsShort: "List a member's versions",
			get: func(client *api.Client, owner, name string) (any, error) {
				return client.GetMember(owner, name)
			},
			listPublic: func(client *api.Client) (any, error) {
				return client.ListPublicMembers()
			},
			listByOwner: func(client *api.Client, owner string) (any, error) {
				return client.ListMembers(owner)
			},
			listVersions: func(client *api.Client, owner, name string) (any, error) {
				return client.ListMemberVersions(owner, name)
			},
		},
		{
			name:          "skill",
			short:         "Show a skill definition",
			listShort:     "List public skills or one owner's skills",
			versionsShort: "List a skill's versions",
			get: func(client *api.Client, owner, name string) (any, error) {
				return client.GetSkill(owner, name)
			},
			listPublic: func(client *api.Client) (any, error) {
				return client.ListPublicSkills()
			},
			listByOwner: func(client *api.Client, owner string) (any, error) {
				return client.ListSkills(owner)
			},
			listVersions: func(client *api.Client, owner, name string) (any, error) {
				return client.ListSkillVersions(owner, name)
			},
		},
		{
			name:          "claude-md",
			short:         "Show a CLAUDE.md definition",
			listShort:     "List public CLAUDE.md files or one owner's files",
			versionsShort: "List a CLAUDE.md file's versions",
			get: func(client *api.Client, owner, name string) (any, error) {
				return client.GetClaudeMd(owner, name)
			},
			listPublic: func(client *api.Client) (any, error) {
				return client.ListPublicClaudeMds()
			},
			listByOwner: func(client *api.Client, owner string) (any, error) {
				return client.ListClaudeMds(owner)
			},
			listVersions: func(client *api.Client, owner, name string) (any, error) {
				return client.ListClaudeMdVersions(owner, name)
			},
		},
		{
			name:          "claude-settings",
			short:         "Show a Claude settings definition",
			listShort:     "List public Claude settings or one owner's settings",
			versionsShort: "List a Claude settings file's versions",
			get: func(client *api.Client, owner, name string) (any, error) {
				return client.GetClaudeSettings(owner, name)
			},
			listPublic: func(client *api.Client) (any, error) {
				return client.ListPublicClaudeSettings()
			},
			listByOwner: func(client *api.Client, owner string) (any, error) {
				return client.ListClaudeSettings(owner)
			},
			listVersions: func(client *api.Client, owner, name string) (any, error) {
				return client.ListClaudeSettingsVersions(owner, name)
			},
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
			item, err := spec.get(client, owner, name)
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
			if len(args) == 0 {
				items, err := spec.listPublic(client)
				if err != nil {
					return err
				}
				return printJSON(items)
			}
			items, err := spec.listByOwner(client, args[0])
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
			items, err := spec.listVersions(client, owner, name)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}
