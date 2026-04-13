package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSkillCmd())
}

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "skill",
		Short:   "Manage agent skills",
		GroupID: rootGroupServer,
		Long:    `Create, edit, and delete reusable agent skills on the server.`,
	}
	cmd.AddCommand(newSkillCreateCmd())
	cmd.AddCommand(newSkillEditCmd())
	cmd.AddCommand(newSkillDeleteCmd())
	return cmd
}

func newSkillCreateCmd() *cobra.Command {
	var name, content, summary string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new skill",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			resp, err := client.CreateResource(api.KindSkill, owner, api.ContentWriteRequest{
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
	cmd.Flags().StringVar(&name, "name", "", "Skill name (lowercase with hyphens)")
	cmd.Flags().StringVar(&content, "content", "", "Skill content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newSkillEditCmd() *cobra.Command {
	var name, content, summary string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Update a skill",
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
			resp, err := client.PatchResource(api.KindSkill, owner, args[0], &body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New skill name")
	cmd.Flags().StringVar(&content, "content", "", "New skill content")
	cmd.Flags().StringVar(&summary, "summary", "", "Short description")
	return cmd
}

func newSkillDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteResource(api.KindSkill, owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
