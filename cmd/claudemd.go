package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newClaudeMdCmd())
}

func newClaudeMdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "claude-md",
		Short:   "Manage CLAUDE.md resources",
		GroupID: rootGroupServer,
		Long:    `Create, edit, and delete CLAUDE.md resources on the server.`,
	}
	cmd.AddCommand(newClaudeMdCreateCmd())
	cmd.AddCommand(newClaudeMdEditCmd())
	cmd.AddCommand(newClaudeMdDeleteCmd())
	return cmd
}

func newClaudeMdCreateCmd() *cobra.Command {
	var name, content, summary string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new CLAUDE.md file",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			resp, err := client.CreateResource(api.KindClaudeMd, owner, api.ContentWriteRequest{
				Name:    name,
				Content: content,
				Summary: summary,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Claude md name")
	cmd.Flags().StringVar(&content, "content", "", "Claude md content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeMdEditCmd() *cobra.Command {
	var name, content, summary string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Update a CLAUDE.md file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			body := api.ContentPatchRequest{}
			if cmd.Flags().Changed("name") {
				body.Name = &name
			}
			if cmd.Flags().Changed("content") {
				body.Content = &content
			}
			if cmd.Flags().Changed("summary") {
				body.Summary = &summary
			}
			resp, err := client.PatchResource(api.KindClaudeMd, owner, args[0], &body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New claude md name")
	cmd.Flags().StringVar(&content, "content", "", "New claude md content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	return cmd
}

func newClaudeMdDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a CLAUDE.md file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteResource(api.KindClaudeMd, owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
