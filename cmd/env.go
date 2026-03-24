package cmd

import (
	"context"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newEnvCmd())
}

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
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
		Use:   "create",
		Short: "Create an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			e, err := domain.NewEnvironment(name, key, value)
			if err != nil {
				return err
			}
			if err := store.CreateEnvironment(context.Background(), e); err != nil {
				return err
			}
			return printJSON(e)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Environment name")
	cmd.Flags().StringVar(&key, "key", "", "Environment variable key")
	cmd.Flags().StringVar(&value, "value", "", "Environment variable value")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("value")
	return cmd
}

func newEnvListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			envs, err := store.ListEnvironments(context.Background())
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
		Use:   "update <id>",
		Short: "Update an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			e, err := store.GetEnvironment(context.Background(), args[0])
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
			if err := store.UpdateEnvironment(context.Background(), &e); err != nil {
				return err
			}
			return printJSON(e)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New environment name")
	cmd.Flags().StringVar(&key, "key", "", "New environment variable key")
	cmd.Flags().StringVar(&value, "value", "", "New environment variable value")
	return cmd
}

func newEnvDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.DeleteEnvironment(context.Background(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
