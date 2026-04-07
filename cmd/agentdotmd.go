package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAgentDotMdCmd())
}

func newAgentDotMdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-dot-md",
		Short: "Manage agent dot md files",
	}
	cmd.AddCommand(newAgentDotMdCreateCmd())
	cmd.AddCommand(newAgentDotMdListCmd())
	cmd.AddCommand(newAgentDotMdUpdateCmd())
	cmd.AddCommand(newAgentDotMdDeleteCmd())
	return cmd
}

func newAgentDotMdCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create an agent dot md file",
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

			c, err := resource.NewAgentDotMd(name, content)
			if err != nil {
				return err
			}
			if err := store.CreateAgentDotMd(cmd.Context(), c); err != nil {
				return err
			}
			return printJSON(c)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Agent dot md name (human identifier)")
	cmd.Flags().StringVar(&content, "content", "", "Agent dot md content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newAgentDotMdListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all agent dot md files",
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

			items, err := store.ListAgentDotMds(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newAgentDotMdUpdateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update an agent dot md file",
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

			c, err := store.GetAgentDotMd(cmd.Context(), args[0])
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
			if err := store.UpdateAgentDotMd(cmd.Context(), &c); err != nil {
				return err
			}
			return printJSON(c)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New agent dot md name")
	cmd.Flags().StringVar(&content, "content", "", "New agent dot md content")
	return cmd
}

func newAgentDotMdDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete an agent dot md file",
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

			if err := store.DeleteAgentDotMd(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
