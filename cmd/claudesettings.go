package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newClaudeSettingsCmd())
}

func newClaudeSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "claude-settings",
		Short:   "Manage Claude settings files",
		GroupID: rootGroupServer,
		Long: `Manage Claude settings files on the server.

Use create, edit, and delete to manage your own files.
Use explore to inspect shared files before you fork or reference them.`,
	}
	cmd.AddCommand(newClaudeSettingsCreateCmd())
	cmd.AddCommand(newClaudeSettingsEditCmd())
	cmd.AddCommand(newClaudeSettingsDeleteCmd())
	return cmd
}

func newClaudeSettingsCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new settings file",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			resp, err := client.CreateClaudeSettings(owner, api.ClaudeSettingsWriteRequest{
				Name:    name,
				Content: content,
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
		Short: "Update a settings file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			current, err := client.GetClaudeSettings(owner, args[0])
			if err != nil {
				return err
			}
			body := api.ClaudeSettingsWriteRequest{
				Name:    current.Name,
				Content: current.Content,
			}
			if cmd.Flags().Changed("name") {
				body.Name = name
			}
			if cmd.Flags().Changed("content") {
				body.Content = content
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
		Short: "Delete a settings file",
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
