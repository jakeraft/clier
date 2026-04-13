package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

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

			apiKind, err := toAPIKind(kind)
			if err != nil {
				return err
			}
			resp, err := client.ForkResource(apiKind, owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}

func toAPIKind(kind string) (api.ResourceKind, error) {
	switch kind {
	case resourceKindMember:
		return api.KindMember, nil
	case resourceKindTeam:
		return api.KindTeam, nil
	case resourceKindSkill:
		return api.KindSkill, nil
	case resourceKindClaudeMd:
		return api.KindClaudeMd, nil
	case resourceKindClaudeSettings:
		return api.KindClaudeSettings, nil
	default:
		return "", errUnsupportedResourceKind(kind)
	}
}
