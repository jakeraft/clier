package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSkillCmd())
}

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage skills",
	}
	cmd.AddCommand(newSkillCreateCmd())
	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillUpdateCmd())
	cmd.AddCommand(newSkillDeleteCmd())
	return cmd
}

func newSkillCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a skill",

		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			resp, err := client.CreateSkill(owner, map[string]string{
				"name":    name,
				"content": content,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Skill name (lowercase with hyphens, e.g. code-review)")
	cmd.Flags().StringVar(&content, "content", "", "Skill content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			items, err := client.ListSkills(owner)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newSkillUpdateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a skill",

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

			resp, err := client.UpdateSkill(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New skill name")
	cmd.Flags().StringVar(&content, "content", "", "New skill content")
	return cmd
}

func newSkillDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a skill",

		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := resolveOwner()

			if err := client.DeleteSkill(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
