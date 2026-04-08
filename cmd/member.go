package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newMemberCmd())
}

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage members",
	}
	cmd.AddCommand(newMemberCreateCmd())
	cmd.AddCommand(newMemberListCmd())
	cmd.AddCommand(newMemberUpdateCmd())
	cmd.AddCommand(newMemberDeleteCmd())
	return cmd
}

func newMemberCreateCmd() *cobra.Command {
	var name, command, claudeMd, claudeSettings, repo string
	var skills []string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a member",
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			body := map[string]any{
				"name":    name,
				"command": command,
			}
			if claudeMd != "" {
				body["claude_md_id"] = claudeMd
			}
			if skills != nil {
				body["skill_ids"] = skills
			}
			if claudeSettings != "" {
				body["claude_settings_id"] = claudeSettings
			}
			if repo != "" {
				body["git_repo_url"] = repo
			}

			resp, err := client.CreateMember(owner, body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Member name")
	cmd.Flags().StringVar(&command, "command", "", "Command (binary + CLI flags, e.g. \"claude --dangerously-skip-permissions\")")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "Claude md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skill IDs (comma-separated)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func newMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all members",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			members, err := client.ListMembers(owner)
			if err != nil {
				return err
			}
			return printJSON(members)
		},
	}
}

func newMemberUpdateCmd() *cobra.Command {
	var name, command, claudeMd, claudeSettings, repo string
	var skills []string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a member",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("command") {
				body["command"] = command
			}
			if cmd.Flags().Changed("claude-md") {
				body["claude_md_id"] = claudeMd
			}
			if cmd.Flags().Changed("skills") {
				body["skill_ids"] = skills
			}
			if cmd.Flags().Changed("claude-settings") {
				body["claude_settings_id"] = claudeSettings
			}
			if cmd.Flags().Changed("repo") {
				body["git_repo_url"] = repo
			}

			resp, err := client.UpdateMember(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New member name")
	cmd.Flags().StringVar(&command, "command", "", "New command (binary + CLI flags)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New agent dot md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "New skill IDs (comma-separated)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL")
	return cmd
}

func newMemberDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a member",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			if err := client.DeleteMember(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
