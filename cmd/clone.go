package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
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
		Long: `Download a team from the server into a local working copy.

Use push/pull to sync changes, and run start to launch agents.

Note: Some vendor CLIs key approvals on the absolute workspace path
(e.g., Codex's directory trust). Cloning under a parent directory
you have already trusted in that vendor avoids one-time approval
prompts on the first start.`,
		GroupID: rootGroupWorkspace,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}

			base, err := resolveCloneBase(resourceTarget{
				Kind:  string(api.KindTeam),
				Owner: owner,
				Name:  name,
			})
			if err != nil {
				return err
			}

			svc := appworkspace.NewService(client, newFileMaterializer(), newGitRepo())
			manifest, err := svc.Clone(base, owner, name)
			if err != nil {
				return err
			}
			return printJSON(map[string]any{
				"status": "cloned",
				"kind":   manifest.Kind,
				"owner":  manifest.Owner,
				"name":   manifest.Name,
				"dir":    base,
				"state":  appworkspace.ManifestPath(base),
			})
		},
	}
}
