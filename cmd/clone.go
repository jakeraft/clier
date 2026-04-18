package cmd

import (
	"errors"
	"fmt"
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCloneCmd())
}

func newCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <owner/name[@version]>",
		Short: "Download a local working copy",
		Long: `Download a team from the server into the canonical workspace
location at <workspace_dir>/<owner>.<name>/.

The workspace directory defaults to ~/.clier/workspace and can be
overridden via the workspace_dir field in ~/.clier/config.json.

Use push/pull to sync changes, and run start to launch agents.

Append @<version> to clone a specific team version.`,
		GroupID: rootGroupWorkspace,
		Args:    requireOneArg("clier clone <owner/name[@version]>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name, version, err := splitVersionedResourceID(args[0])
			if err != nil {
				return err
			}
			if err := validateOwner(owner); err != nil {
				return err
			}

			base := workingCopyPath(owner, name)
			if _, err := os.Stat(base); err == nil {
				return &domain.Fault{
					Kind: domain.KindCloneDestExists,
					Subject: map[string]string{
						"path":  base,
						"owner": owner,
						"name":  name,
					},
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("stat clone destination: %w", err)
			}

			svc := appworkspace.NewService(client, newFileMaterializer(), newGitRepo())
			var manifest *appworkspace.Manifest
			if version != nil {
				manifest, err = svc.CloneVersion(base, owner, name, *version)
			} else {
				manifest, err = svc.Clone(base, owner, name)
			}
			if err != nil {
				return err
			}
			return printJSON(cloneResultPayload(base, manifest))
		},
	}
}
