package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/app/tutorial"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tutorial",
		Short: "Set up tutorial scenarios with sample data",
	}
	cmd.AddCommand(newTutorialListCmd())
	cmd.AddCommand(newTutorialRunCmd())
	cmd.AddCommand(newTutorialCleanCmd())
	return cmd
}

func newTutorialListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available tutorial scenarios",
		RunE: func(cmd *cobra.Command, args []string) error {
			scenarios := tutorial.List()
			if len(scenarios) == 0 {
				fmt.Println("No scenarios available.")
				return nil
			}
			for _, s := range scenarios {
				fmt.Printf("%-20s %s\n", s.Name, s.Description)
			}
			return nil
		},
	}
}

func newTutorialRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "run <scenario>",
		Short:       "Run a tutorial scenario (clean + create)",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scenario, err := tutorial.Get(args[0])
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := tutorial.Run(cmd.Context(), store, scenario); err != nil {
				return err
			}

			fmt.Printf("Scenario %q created successfully.\n", scenario.Name)
			return nil
		},
	}
}

func newTutorialCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "clean <scenario>",
		Short:       "Delete tutorial scenario data",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scenario, err := tutorial.Get(args[0])
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := tutorial.Clean(cmd.Context(), store, scenario); err != nil {
				return err
			}

			fmt.Printf("Scenario %q cleaned.\n", scenario.Name)
			return nil
		},
	}
}
