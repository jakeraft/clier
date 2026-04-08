package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newClaudeMdCmd())
}

func newClaudeMdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude-md",
		Short: "Manage claude md files",
	}
	cmd.AddCommand(newClaudeMdCreateCmd())
	cmd.AddCommand(newClaudeMdListCmd())
	cmd.AddCommand(newClaudeMdUpdateCmd())
	cmd.AddCommand(newClaudeMdDeleteCmd())
	return cmd
}

func newClaudeMdCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a claude md file",

		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			resp, err := client.CreateClaudeMd(owner, map[string]string{
				"name":    name,
				"content": content,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Claude md name (human identifier)")
	cmd.Flags().StringVar(&content, "content", "", "Claude md content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeMdListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all claude md files",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			items, err := client.ListClaudeMds(owner)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newClaudeMdUpdateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "update <name>",
		Short:       "Update a claude md file by name",

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

			resp, err := client.UpdateClaudeMd(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New claude md name")
	cmd.Flags().StringVar(&content, "content", "", "New claude md content")
	return cmd
}

func newClaudeMdDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <name>",
		Short:       "Delete a claude md file by name",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			if err := client.DeleteClaudeMd(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
