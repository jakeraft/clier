package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
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
		Use:         "create",
		Short:       "Create a CLI profile",
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

			p, err := resource.NewCliProfile(name, preset, customArgs)
			if err != nil {
				return err
			}
			if err := store.CreateCliProfile(cmd.Context(), p); err != nil {
				return err
			}
			return printJSON(p)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Profile name")
	cmd.Flags().StringVar(&preset, "preset", "", "Preset key (e.g. claude-sonnet, claude-opus)")
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

			profiles, err := store.ListCliProfiles(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(profiles)
		},
	}
}

func newProfileUpdateCmd() *cobra.Command {
	var name string
	var customArgs []string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update a CLI profile",
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

			p, err := store.GetCliProfile(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var customArgsPtr *[]string
			if cmd.Flags().Changed("args") {
				customArgsPtr = &customArgs
			}

			if err := p.Update(namePtr, customArgsPtr); err != nil {
				return err
			}
			if err := store.UpdateCliProfile(cmd.Context(), &p); err != nil {
				return err
			}
			return printJSON(p)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New profile name")
	cmd.Flags().StringSliceVar(&customArgs, "args", nil, "New custom CLI arguments (comma-separated)")
	return cmd
}

func newProfileDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a CLI profile",
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

			if err := store.DeleteCliProfile(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
