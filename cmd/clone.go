package cmd

import (
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCloneCmd())
}

func newCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "clone <[owner/]name>",
		Short:   "Clone a resource into a local clone",
		GroupID: rootGroupRuntime,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			kind, err := resolveServerResourceKind(client, owner, name)
			if err != nil {
				return err
			}

			base, err := resolveCloneBase(resourceTarget{
				Kind:  kind,
				Owner: owner,
				Name:  name,
			})
			if err != nil {
				return err
			}

			svc := appworkspace.NewService(client)
			manifest, err := cloneResolvedResource(svc, base, kind, owner, name)
			if err != nil {
				return err
			}
			return printJSON(map[string]any{
				"status":   "cloned",
				"kind":     manifest.Kind,
				"owner":    manifest.Owner,
				"name":     manifest.Name,
				"dir":      base,
				"manifest": appworkspace.ManifestPath(base),
			})
		},
	}
}

func cloneResolvedResource(svc *appworkspace.Service, base, kind, owner, name string) (*appworkspace.Manifest, error) {
	switch kind {
	case resourceKindMember:
		return svc.CloneMember(base, owner, name)
	case resourceKindTeam:
		return svc.CloneTeam(base, owner, name)
	default:
		return nil, errUnsupportedResourceKind(kind)
	}
}
