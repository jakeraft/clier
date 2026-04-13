package cmd

import (
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
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
		Long: `Clone a resource from the server into a local working copy.
This creates a directory with editable projections and materialized files.
Use push/pull to sync changes, and run start to launch agents.`,
		GroupID: rootGroupWorkspace,
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

			svc := appworkspace.NewService(client, newFileMaterializer(), newGitRepo())
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
	case string(api.KindMember):
		return svc.CloneMember(base, owner, name)
	case string(api.KindTeam):
		return svc.CloneTeam(base, owner, name)
	default:
		return nil, fmt.Errorf("unsupported resource kind %q", kind)
	}
}
