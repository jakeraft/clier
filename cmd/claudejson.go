package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newClaudeJsonCmd())
}

func newClaudeJsonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude-json",
		Short: "Manage .claude.json files",
	}
	cmd.AddCommand(newClaudeJsonCreateCmd())
	cmd.AddCommand(newClaudeJsonListCmd())
	cmd.AddCommand(newClaudeJsonUpdateCmd())
	cmd.AddCommand(newClaudeJsonDeleteCmd())
	return cmd
}

func newClaudeJsonCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a .claude.json file",
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

			c, err := resource.NewClaudeJson(name, content)
			if err != nil {
				return err
			}
			if err := store.CreateClaudeJson(cmd.Context(), c); err != nil {
				return err
			}
			return printJSON(c)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", ".claude.json name (human identifier)")
	cmd.Flags().StringVar(&content, "content", "", ".claude.json content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeJsonListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all .claude.json files",
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

			items, err := store.ListClaudeJsons(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newClaudeJsonUpdateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a .claude.json file",
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

			c, err := store.GetClaudeJson(cmd.Context(), args[0])
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
			if err := store.UpdateClaudeJson(cmd.Context(), &c); err != nil {
				return err
			}
			return printJSON(c)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New .claude.json name")
	cmd.Flags().StringVar(&content, "content", "", "New .claude.json content")
	return cmd
}

func newClaudeJsonDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a .claude.json file",
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

			if err := store.DeleteClaudeJson(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
