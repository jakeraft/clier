package cmd

import (
	"fmt"
	"os"

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
		Long: `Download a team from the server into the canonical workspace
location at <workspace_dir>/<owner>/<name>/.

The workspace directory defaults to ~/.clier/workspace and can be
overridden via the workspace_dir field in ~/.clier/config.json.

Use push/pull to sync changes, and run start to launch agents.`,
		GroupID: rootGroupWorkspace,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}
			if err := validateOwner(owner); err != nil {
				return err
			}

			base := workingCopyPath(owner, name)
			if _, err := os.Stat(base); err == nil {
				return fmt.Errorf("clone destination already exists: %s", base)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat clone destination: %w", err)
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
