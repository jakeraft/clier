package cmd

import (
	"context"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newPromptCmd())
}

func newPromptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Manage system prompts",
	}
	cmd.AddCommand(newPromptCreateCmd())
	cmd.AddCommand(newPromptListCmd())
	cmd.AddCommand(newPromptUpdateCmd())
	cmd.AddCommand(newPromptDeleteCmd())
	return cmd
}

func newPromptCreateCmd() *cobra.Command {
	var name, prompt string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a system prompt",
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

			s, err := domain.NewSystemPrompt(name, prompt)
			if err != nil {
				return err
			}
			if err := store.CreateSystemPrompt(context.Background(), s); err != nil {
				return err
			}
			return printJSON(s)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Prompt name")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt text")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("prompt")
	return cmd
}

func newPromptListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all system prompts",
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

			prompts, err := store.ListSystemPrompts(context.Background())
			if err != nil {
				return err
			}
			return printJSON(prompts)
		},
	}
}

func newPromptUpdateCmd() *cobra.Command {
	var name, prompt string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a system prompt",
		Args:  cobra.ExactArgs(1),
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

			s, err := store.GetSystemPrompt(context.Background(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var promptPtr *string
			if cmd.Flags().Changed("prompt") {
				promptPtr = &prompt
			}

			if err := s.Update(namePtr, promptPtr); err != nil {
				return err
			}
			if err := store.UpdateSystemPrompt(context.Background(), &s); err != nil {
				return err
			}
			return printJSON(s)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New prompt name")
	cmd.Flags().StringVar(&prompt, "prompt", "", "New prompt text")
	return cmd
}

func newPromptDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a system prompt",
		Args:  cobra.ExactArgs(1),
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

			if err := store.DeleteSystemPrompt(context.Background(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
