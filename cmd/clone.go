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
		Use:   "clone <owner/name>",
		Short: "Download a local working copy",
		Long: `Download a resource from the server into a local working copy.
Works with both teams and individual members — cloning a member
automatically creates a runnable 1-member workspace.

Use push/pull to sync changes, and run start to launch agents.`,
		GroupID: rootGroupWorkspace,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, err := parseOwnerName(args[0])
			if err != nil {
				return err
			}
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
			manifest, err := svc.Clone(base, kind, owner, name)
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
