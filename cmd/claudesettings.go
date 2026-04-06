package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newClaudeSettingsCmd())
}

func newClaudeSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude-settings",
		Short: "Manage Claude settings.json files",
	}
	cmd.AddCommand(newClaudeSettingsCreateCmd())
	cmd.AddCommand(newClaudeSettingsListCmd())
	cmd.AddCommand(newClaudeSettingsUpdateCmd())
	cmd.AddCommand(newClaudeSettingsDeleteCmd())
	return cmd
}

func newClaudeSettingsCreateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a Claude settings.json file",
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

			s, err := resource.NewSettings(name, content)
			if err != nil {
				return err
			}
			if err := store.CreateSettings(cmd.Context(), s); err != nil {
				return err
			}
			return printJSON(s)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Settings name (human identifier)")
	cmd.Flags().StringVar(&content, "content", "", "Settings JSON content")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newClaudeSettingsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all Claude settings.json files",
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

			items, err := store.ListSettings(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(items)
		},
	}
}

func newClaudeSettingsUpdateCmd() *cobra.Command {
	var name, content string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a Claude settings.json file",
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

			s, err := store.GetSettings(cmd.Context(), args[0])
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
			if err := store.UpdateSettings(cmd.Context(), &s); err != nil {
				return err
			}
			return printJSON(s)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New settings name")
	cmd.Flags().StringVar(&content, "content", "", "New settings JSON content")
	return cmd
}

func newClaudeSettingsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a Claude settings.json file",
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

			if err := store.DeleteSettings(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
