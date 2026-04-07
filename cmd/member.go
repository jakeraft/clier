package cmd

import (
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newMemberCmd())
}

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage members",
	}
	cmd.AddCommand(newMemberCreateCmd())
	cmd.AddCommand(newMemberListCmd())
	cmd.AddCommand(newMemberUpdateCmd())
	cmd.AddCommand(newMemberDeleteCmd())
	return cmd
}

func newMemberCreateCmd() *cobra.Command {
	var name, model, claudeMd, settings, claudeJson, repo string
	var cliArgs, skills []string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a member",
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

			m, err := domain.NewMember(name, model, cliArgs, claudeMd, skills, settings, claudeJson, repo)
			if err != nil {
				return err
			}
			if err := store.CreateMember(cmd.Context(), m); err != nil {
				return err
			}
			return printJSON(m)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Member name")
	cmd.Flags().StringVar(&model, "model", "", "Model identifier")
	cmd.Flags().StringSliceVar(&cliArgs, "args", nil, "CLI arguments (comma-separated)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "CLAUDE.md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skill IDs (comma-separated)")
	cmd.Flags().StringVar(&settings, "settings", "", "Settings resource ID")
	cmd.Flags().StringVar(&claudeJson, "claude-json", "", "ClaudeJson resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo ID")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("model")
	return cmd
}

func newMemberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all members",
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

			members, err := store.ListMembers(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(members)
		},
	}
}

func newMemberUpdateCmd() *cobra.Command {
	var name, model, claudeMd, settings, claudeJson, repo string
	var cliArgs, skills []string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a member",
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

			m, err := store.GetMember(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var modelPtr *string
			if cmd.Flags().Changed("model") {
				modelPtr = &model
			}
			var argsPtr *[]string
			if cmd.Flags().Changed("args") {
				argsPtr = &cliArgs
			}
			var claudeMdPtr *string
			if cmd.Flags().Changed("claude-md") {
				claudeMdPtr = &claudeMd
			}
			var skillsPtr *[]string
			if cmd.Flags().Changed("skills") {
				skillsPtr = &skills
			}
			var settingsPtr *string
			if cmd.Flags().Changed("settings") {
				settingsPtr = &settings
			}
			var claudeJsonPtr *string
			if cmd.Flags().Changed("claude-json") {
				claudeJsonPtr = &claudeJson
			}
			var repoPtr *string
			if cmd.Flags().Changed("repo") {
				repoPtr = &repo
			}

			if err := m.Update(namePtr, modelPtr, argsPtr, claudeMdPtr, skillsPtr, settingsPtr, claudeJsonPtr, repoPtr); err != nil {
				return err
			}
			if err := store.UpdateMember(cmd.Context(), &m); err != nil {
				return err
			}
			return printJSON(m)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New member name")
	cmd.Flags().StringVar(&model, "model", "", "New model identifier")
	cmd.Flags().StringSliceVar(&cliArgs, "args", nil, "New CLI arguments (comma-separated)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New CLAUDE.md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "New skill IDs (comma-separated)")
	cmd.Flags().StringVar(&settings, "settings", "", "New settings resource ID")
	cmd.Flags().StringVar(&claudeJson, "claude-json", "", "New ClaudeJson resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo ID")
	return cmd
}

func newMemberDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a member",
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

			if err := store.DeleteMember(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
