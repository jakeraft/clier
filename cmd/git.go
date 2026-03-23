package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/settings"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(gitCmd)
}

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Manage git credentials",
}

func init() {
	gitCmd.AddCommand(gitSetCmd)
	gitCmd.AddCommand(gitGetCmd)
	gitCmd.AddCommand(gitRemoveCmd)
	gitCmd.AddCommand(gitListCmd)
}

var gitSetCmd = &cobra.Command{
	Use:   "set <host> <token>",
	Short: "Set git credential for a host",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := settings.New()
		if err != nil {
			return err
		}
		if err := s.SetCredential(args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "credential set for %s\n", args[0])
		return nil
	},
}

var gitGetCmd = &cobra.Command{
	Use:   "get <host>",
	Short: "Get git credential for a host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := settings.New()
		if err != nil {
			return err
		}
		token, err := s.GetCredential(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), token)
		return nil
	},
}

var gitRemoveCmd = &cobra.Command{
	Use:   "remove <host>",
	Short: "Remove git credential for a host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := settings.New()
		if err != nil {
			return err
		}
		if err := s.RemoveCredential(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "credential removed for %s\n", args[0])
		return nil
	},
}

var gitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all git credential hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := settings.New()
		if err != nil {
			return err
		}
		hosts, err := s.ListCredentialHosts()
		if err != nil {
			return err
		}
		if len(hosts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no credentials configured")
			return nil
		}
		for _, h := range hosts {
			fmt.Fprintln(cmd.OutOrStdout(), h)
		}
		return nil
	},
}
