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
	cmd.AddCommand(newClaudeMdListCmd())
	cmd.AddCommand(newClaudeMdViewCmd())
	cmd.AddCommand(newClaudeMdCreateCmd())
	cmd.AddCommand(newClaudeMdEditCmd())
	cmd.AddCommand(newClaudeMdDeleteCmd())
	cmd.AddCommand(newClaudeMdForkCmd())
	return cmd
}

func newClaudeMdListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [owner]",
		Short: "List claude-md files",
		Long:  "List your claude-md files, or another user's if [owner] is given.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			var owner string
			if len(args) == 1 {
				owner = args[0]
			} else {
				owner = requireLogin()
			}
			items, err := client.ListClaudeMds(owner)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newClaudeMdViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <[owner/]name>",
		Short: "View a claude-md file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			item, err := client.GetClaudeMd(owner, name)
			if err != nil {
				return err
			}
			return printJSON(item)
		},
	}
}

func newClaudeMdCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a claude-md file",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
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
	cmd.Flags().StringVar(&name, "name", "", "Claude md name")
	cmd.Flags().StringVar(&content, "content", "", "Claude md content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeMdEditCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a claude-md file",
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
		Use:   "delete <name>",
		Short: "Delete a claude-md file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteClaudeMd(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newClaudeMdForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <owner/name>",
		Short: "Fork a claude-md file to your namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			owner, name := parseOwnerName(args[0])
			resp, err := client.ForkClaudeMd(owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}
