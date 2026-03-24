package cmd

import (
	"context"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newProfileCmd())
}

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage CLI profiles",
	}
	cmd.AddCommand(newProfileCreateCmd())
	cmd.AddCommand(newProfileListCmd())
	cmd.AddCommand(newProfileUpdateCmd())
	cmd.AddCommand(newProfileDeleteCmd())
	return cmd
}

func newProfileCreateCmd() *cobra.Command {
	var name, preset string
	var customArgs []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a CLI profile",
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

			p, err := domain.NewCliProfile(name, preset, customArgs)
			if err != nil {
				return err
			}
			if err := store.CreateCliProfile(context.Background(), p); err != nil {
				return err
			}
			return printJSON(p)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Profile name")
	cmd.Flags().StringVar(&preset, "preset", "", "Preset key (e.g. claude-sonnet, codex-mini)")
	cmd.Flags().StringSliceVar(&customArgs, "args", nil, "Custom CLI arguments (comma-separated)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("preset")
	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all CLI profiles",
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

			profiles, err := store.ListCliProfiles(context.Background())
			if err != nil {
				return err
			}
			return printJSON(profiles)
		},
	}
}

func newProfileUpdateCmd() *cobra.Command {
	var name, customArgsStr string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a CLI profile",
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

			p, err := store.GetCliProfile(context.Background(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var customArgsPtr *[]string
			if cmd.Flags().Changed("args") {
				parts := strings.Split(customArgsStr, ",")
				customArgsPtr = &parts
			}

			if err := p.Update(namePtr, customArgsPtr); err != nil {
				return err
			}
			if err := store.UpdateCliProfile(context.Background(), &p); err != nil {
				return err
			}
			return printJSON(p)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New profile name")
	cmd.Flags().StringVar(&customArgsStr, "args", "", "New custom CLI arguments (comma-separated)")
	return cmd
}

func newProfileDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a CLI profile",
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

			if err := store.DeleteCliProfile(context.Background(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
