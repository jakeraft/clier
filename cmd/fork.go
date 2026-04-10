package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(newForkCmd())
}

func newForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "fork <owner/name>",
		Short:   "Fork a public resource into your namespace",
		GroupID: rootGroupRuntime,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			owner, name, err := parseExplicitOwnerName(args[0])
			if err != nil {
				return err
			}
			kind, err := resolveServerResourceKind(client, owner, name)
			if err != nil {
				return err
			}

			switch kind {
			case resourceKindMember:
				resp, err := client.ForkMember(owner, name)
				if err != nil {
					return err
				}
				return printJSON(resp)
			case resourceKindTeam:
				resp, err := client.ForkTeam(owner, name)
				if err != nil {
					return err
				}
				return printJSON(resp)
			default:
				return errUnsupportedResourceKind(kind)
			}
		},
	}
}
