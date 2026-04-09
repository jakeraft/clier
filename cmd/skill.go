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
		Short:   "Manage skills",
		GroupID: rootGroupServer,
	}
	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillViewCmd())
	cmd.AddCommand(newSkillCreateCmd())
	cmd.AddCommand(newSkillEditCmd())
	cmd.AddCommand(newSkillDeleteCmd())
	cmd.AddCommand(newSkillForkCmd())
	return cmd
}

func newSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [owner]",
		Short: "List skills",
		Long:  "List your skills, or another user's skills if [owner] is given.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			var owner string
			if len(args) == 1 {
				owner = args[0]
			} else {
				owner = requireLogin()
			}
			items, err := client.ListSkills(owner)
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newSkillViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <[owner/]name>",
		Short: "View a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			item, err := client.GetSkill(owner, name)
			if err != nil {
				return err
			}
			return printJSON(item)
		},
	}
}

func newSkillCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a skill",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			resp, err := client.CreateSkill(owner, api.SkillMutationRequest{
				Name:    name,
				Content: content,
			})
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Skill name (lowercase with hyphens)")
	cmd.Flags().StringVar(&content, "content", "", "Skill content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newSkillEditCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			current, err := client.GetSkill(owner, args[0])
			if err != nil {
				return err
			}
			body := api.SkillMutationRequest{
				Name:    current.Name,
				Content: current.Content,
			}
			if cmd.Flags().Changed("name") {
				body.Name = name
			}
			if cmd.Flags().Changed("content") {
				body.Content = content
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
		Use:   "delete <name>",
		Short: "Delete a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteSkill(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newSkillForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <owner/name>",
		Short: "Fork a skill to your namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			owner, name := parseOwnerName(args[0])
			resp, err := client.ForkSkill(owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}
