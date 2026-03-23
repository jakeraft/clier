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
	var name, profile, repo string
	var prompts []string
	var envs []string

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

			m, err := domain.NewMember(name, profile, prompts, repo, envs)
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
	cmd.Flags().StringVar(&profile, "profile", "", "CLI profile ID")
	cmd.Flags().StringSliceVar(&prompts, "prompts", nil, "System prompt IDs (comma-separated)")
	cmd.Flags().StringVar(&repo, "repo", "", "Git repo ID")
	cmd.Flags().StringSliceVar(&envs, "envs", nil, "Env IDs (comma-separated)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("profile")
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
	var name, profile, repo string
	var prompts []string
	var envs []string

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
			var profilePtr *string
			if cmd.Flags().Changed("profile") {
				profilePtr = &profile
			}
			var promptIDsPtr *[]string
			if cmd.Flags().Changed("prompts") {
				promptIDsPtr = &prompts
			}
			var repoPtr *string
			if cmd.Flags().Changed("repo") {
				repoPtr = &repo
			}
			var envIDsPtr *[]string
			if cmd.Flags().Changed("envs") {
				envIDsPtr = &envs
			}

			if err := m.Update(namePtr, profilePtr, promptIDsPtr, repoPtr, envIDsPtr); err != nil {
				return err
			}
			if err := store.UpdateMember(cmd.Context(), &m); err != nil {
				return err
			}
			return printJSON(m)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New member name")
	cmd.Flags().StringVar(&profile, "profile", "", "New CLI profile ID")
	cmd.Flags().StringSliceVar(&prompts, "prompts", nil, "New system prompt IDs (comma-separated)")
	cmd.Flags().StringVar(&repo, "repo", "", "New git repo ID")
	cmd.Flags().StringSliceVar(&envs, "envs", nil, "New env IDs (comma-separated)")
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
