package cmd

import (
	"github.com/jakeraft/clier/internal/domain/resource"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newEnvCmd())
}

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environment variables",
	}
	cmd.AddCommand(newEnvCreateCmd())
	cmd.AddCommand(newEnvListCmd())
	cmd.AddCommand(newEnvUpdateCmd())
	cmd.AddCommand(newEnvDeleteCmd())
	return cmd
}

func newEnvCreateCmd() *cobra.Command {
	var name, key, value string

	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create an environment variable",
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

			e, err := resource.NewEnv(name, key, value)
			if err != nil {
				return err
			}
			if err := store.CreateEnv(cmd.Context(), e); err != nil {
				return err
			}
			return printJSON(e)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Env name (human identifier)")
	cmd.Flags().StringVar(&key, "key", "", "Environment variable key")
	cmd.Flags().StringVar(&value, "value", "", "Environment variable value")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("key")
	return cmd
}

func newEnvListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all environment variables",
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

			envs, err := store.ListEnvs(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(envs)
		},
	}
}

func newEnvUpdateCmd() *cobra.Command {
	var name, key, value string

	cmd := &cobra.Command{
		Use:         "update <id>",
		Short:       "Update an environment variable",
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

			e, err := store.GetEnv(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			var namePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			var keyPtr *string
			if cmd.Flags().Changed("key") {
				keyPtr = &key
			}
			var valuePtr *string
			if cmd.Flags().Changed("value") {
				valuePtr = &value
			}

			if err := e.Update(namePtr, keyPtr, valuePtr); err != nil {
				return err
			}
			if err := store.UpdateEnv(cmd.Context(), &e); err != nil {
				return err
			}
			return printJSON(e)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New env name")
	cmd.Flags().StringVar(&key, "key", "", "New env key")
	cmd.Flags().StringVar(&value, "value", "", "New env value")
	return cmd
}

func newEnvDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete an environment variable",
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

			if err := store.DeleteEnv(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
