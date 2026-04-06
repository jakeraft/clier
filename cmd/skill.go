package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
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
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			s, err := resource.NewSkill(name, content)
			if err != nil {
				return err
			}
			if err := store.CreateSkill(cmd.Context(), s); err != nil {
				return err
			}
			return printJSON(s)
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
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			items, err := store.ListSkills(cmd.Context())
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
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			s, err := store.GetSkill(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var contentPtr *string
			if cmd.Flags().Changed("content") {
				contentPtr = &content
			}

			if err := s.Update(namePtr, contentPtr); err != nil {
				return err
			}
			if err := store.UpdateSkill(cmd.Context(), &s); err != nil {
				return err
			}
			return printJSON(s)
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
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.DeleteSkill(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
