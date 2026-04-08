package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
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
	cmd.AddCommand(newClaudeMdCreateCmd())
	cmd.AddCommand(newClaudeMdListCmd())
	cmd.AddCommand(newClaudeMdUpdateCmd())
	cmd.AddCommand(newClaudeMdDeleteCmd())
	return cmd
}

func newClaudeMdCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a claude md file",
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

			c, err := resource.NewClaudeMd(name, content)
			if err != nil {
				return err
			}
			if err := store.CreateClaudeMd(cmd.Context(), c); err != nil {
				return err
			}
			return printJSON(c)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Claude md name (human identifier)")
	cmd.Flags().StringVar(&content, "content", "", "Claude md content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeMdListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all claude md files",
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

			items, err := store.ListClaudeMds(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newClaudeMdUpdateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a claude md file",
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

			c, err := store.GetClaudeMd(cmd.Context(), args[0])
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

			if err := c.Update(namePtr, contentPtr); err != nil {
				return err
			}
			if err := store.UpdateClaudeMd(cmd.Context(), &c); err != nil {
				return err
			}
			return printJSON(c)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New claude md name")
	cmd.Flags().StringVar(&content, "content", "", "New claude md content")
	return cmd
}

func newClaudeMdDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a claude md file",
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

			if err := store.DeleteClaudeMd(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
