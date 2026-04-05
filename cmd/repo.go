package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRepoCmd())
}

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage git repos",
	}
	cmd.AddCommand(newRepoCreateCmd())
	cmd.AddCommand(newRepoListCmd())
	cmd.AddCommand(newRepoUpdateCmd())
	cmd.AddCommand(newRepoDeleteCmd())
	return cmd
}

func newRepoCreateCmd() *cobra.Command {
	var name, url string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a git repo",
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

			r, err := resource.NewGitRepo(name, url)
			if err != nil {
				return err
			}
			if err := store.CreateGitRepo(cmd.Context(), r); err != nil {
				return err
			}
			return printJSON(r)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Repo name")
	cmd.Flags().StringVar(&url, "url", "", "Repo URL")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newRepoListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all git repos",
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

			repos, err := store.ListGitRepos(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(repos)
		},
	}
}

func newRepoUpdateCmd() *cobra.Command {
	var name, url string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a git repo",
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

			r, err := store.GetGitRepo(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var urlPtr *string
			if cmd.Flags().Changed("url") {
				urlPtr = &url
			}

			if err := r.Update(namePtr, urlPtr); err != nil {
				return err
			}
			if err := store.UpdateGitRepo(cmd.Context(), &r); err != nil {
				return err
			}
			return printJSON(r)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New repo name")
	cmd.Flags().StringVar(&url, "url", "", "New repo URL")
	return cmd
}

func newRepoDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a git repo",
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

			if err := store.DeleteGitRepo(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
