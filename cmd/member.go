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
	var name, command, claudeMd, claudeSettings, repo string
	var skills []string

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

			m, err := domain.NewMember(name, command, claudeMd, skills, claudeSettings, repo)
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
	cmd.Flags().StringVar(&command, "command", "", "Command (binary + CLI flags, e.g. \"claude --dangerously-skip-permissions\")")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "Claude md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skill IDs (comma-separated)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("command")
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
	var name, command, claudeMd, claudeSettings, repo string
	var skills []string

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
			var commandPtr *string
			if cmd.Flags().Changed("command") {
				commandPtr = &command
			}
			var claudeMdPtr *string
			if cmd.Flags().Changed("claude-md") {
				claudeMdPtr = &claudeMd
			}
			var skillsPtr *[]string
			if cmd.Flags().Changed("skills") {
				skillsPtr = &skills
			}
			var claudeSettingsPtr *string
			if cmd.Flags().Changed("claude-settings") {
				claudeSettingsPtr = &claudeSettings
			}
			var repoPtr *string
			if cmd.Flags().Changed("repo") {
				repoPtr = &repo
			}

			if err := m.Update(namePtr, commandPtr, claudeMdPtr, skillsPtr, claudeSettingsPtr, repoPtr); err != nil {
				return err
			}
			if err := store.UpdateMember(cmd.Context(), &m); err != nil {
				return err
			}
			return printJSON(m)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New member name")
	cmd.Flags().StringVar(&command, "command", "", "New command (binary + CLI flags)")
	cmd.Flags().StringVar(&claudeMd, "claude-md", "", "New agent dot md resource ID")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "New skill IDs (comma-separated)")
	cmd.Flags().StringVar(&claudeSettings, "claude-settings", "", "New Claude settings resource ID")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL")
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
