package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newClaudeSettingsCmd())
}

func newClaudeSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude-settings",
		Short: "Manage Claude settings.json files",
	}
	cmd.AddCommand(newClaudeSettingsListCmd())
	cmd.AddCommand(newClaudeSettingsViewCmd())
	cmd.AddCommand(newClaudeSettingsCreateCmd())
	cmd.AddCommand(newClaudeSettingsEditCmd())
	cmd.AddCommand(newClaudeSettingsDeleteCmd())
	cmd.AddCommand(newClaudeSettingsForkCmd())
	return cmd
}

func newClaudeSettingsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [owner]",
		Short: "List claude-settings files",
		Long:  "List your claude-settings files, or another user's if [owner] is given.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			var owner string
			if len(args) == 1 {
				owner = args[0]
			} else {
				owner = requireLogin()
			}
			items, err := client.ListClaudeSettings(owner)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newClaudeSettingsViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <[owner/]name>",
		Short: "View a claude-settings file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			item, err := client.GetClaudeSettings(owner, name)
			if err != nil {
				return err
			}
			return printJSON(item)
		},
	}
}

func newClaudeSettingsCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a claude-settings file",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			resp, err := client.CreateClaudeSettings(owner, map[string]string{
				"name":    name,
				"content": content,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Settings name")
	cmd.Flags().StringVar(&content, "content", "", "Settings JSON content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeSettingsEditCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a claude-settings file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()

			body := map[string]string{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("content") {
				body["content"] = content
			}

			resp, err := client.UpdateClaudeSettings(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New settings name")
	cmd.Flags().StringVar(&content, "content", "", "New settings JSON content")
	return cmd
}

func newClaudeSettingsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a claude-settings file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteClaudeSettings(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newClaudeSettingsForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <owner/name>",
		Short: "Fork a claude-settings file to your namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			owner, name := parseOwnerName(args[0])
			resp, err := client.ForkClaudeSettings(owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}
