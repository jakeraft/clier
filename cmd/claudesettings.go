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
	cmd.AddCommand(newClaudeSettingsCreateCmd())
	cmd.AddCommand(newClaudeSettingsListCmd())
	cmd.AddCommand(newClaudeSettingsUpdateCmd())
	cmd.AddCommand(newClaudeSettingsDeleteCmd())
	return cmd
}

func newClaudeSettingsCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a Claude settings.json file",

		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

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
	cmd.Flags().StringVar(&name, "name", "", "Settings name (human identifier)")
	cmd.Flags().StringVar(&content, "content", "", "Settings JSON content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeSettingsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all Claude settings.json files",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			items, err := client.ListClaudeSettings(owner)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newClaudeSettingsUpdateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "update <name>",
		Short:       "Update a Claude settings.json file by name",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

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
		Use:         "delete <name>",
		Short:       "Delete a Claude settings.json file by name",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			if err := client.DeleteClaudeSettings(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
