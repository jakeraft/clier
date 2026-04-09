package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newExploreCmd())
}

func newExploreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "explore",
		Short:   "Discover public resources",
		GroupID: rootGroupDiscovery,
	}
	cmd.AddCommand(newExploreTeamsCmd())
	cmd.AddCommand(newExploreMembersCmd())
	cmd.AddCommand(newExploreSkillsCmd())
	cmd.AddCommand(newExploreClaudeMdsCmd())
	cmd.AddCommand(newExploreClaudeSettingsCmd())
	return cmd
}

func newExploreTeamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teams",
		Short: "List all public teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			items, err := client.ListPublicTeams()
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newExploreMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members",
		Short: "List all public members",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			items, err := client.ListPublicMembers()
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newExploreSkillsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skills",
		Short: "List all public skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			items, err := client.ListPublicSkills()
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newExploreClaudeMdsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "claude-mds",
		Short: "List all public claude-md files",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			items, err := client.ListPublicClaudeMds()
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newExploreClaudeSettingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "claude-settings",
		Short: "List all public claude-settings files",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			items, err := client.ListPublicClaudeSettings()
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}
