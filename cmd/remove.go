package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	apprun "github.com/jakeraft/clier/internal/app/run"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRemoveCmd())
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <owner/name>",
		Short: "Remove a working copy and its run plans",
		Long: `Remove the working copy at <workspace_dir>/<owner>/<name>/
together with every run plan in <workspace_dir>/.runs/ that points
to it.

remove is the symmetric counterpart of clone — using it instead of
"rm -rf" keeps run lists clean and prevents orphan plans.

Refused when:
  - the working copy has uncommitted changes (push or revert first)
  - any associated run plan is still running (clier run stop first)`,
		GroupID: rootGroupWorkspace,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}
			if err := validateOwner(owner); err != nil {
				return err
			}
			base := workingCopyPath(owner, name)

			fs := newFileMaterializer()
			if _, err := appworkspace.LoadManifest(fs, base); err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}

			svc := appworkspace.NewService(newAPIClient(), fs, newGitRepo())
			modified, err := svc.ModifiedTrackedResources(base)
			if err != nil {
				return err
			}
			if len(modified) > 0 {
				return fmt.Errorf("working copy has uncommitted changes; push or revert before removing (%d modified)", len(modified))
			}

			plans, err := apprun.ListPlans(runsDir())
			if err != nil {
				return err
			}
			var owned []*apprun.RunPlan
			for _, p := range plans {
				if p.WorkingCopyPath == base {
					owned = append(owned, p)
				}
			}
			for _, p := range owned {
				if p.Status == apprun.StatusRunning {
					return fmt.Errorf("run %s is still running; clier run stop %s before removing", p.RunID, p.RunID)
				}
			}

			removedRuns := make([]string, 0, len(owned))
			for _, p := range owned {
				if err := os.Remove(apprun.PlanPath(runsDir(), p.RunID)); err != nil {
					return fmt.Errorf("remove run plan %s: %w", p.RunID, err)
				}
				removedRuns = append(removedRuns, p.RunID)
			}
			if err := os.RemoveAll(base); err != nil {
				return fmt.Errorf("remove working copy %s: %w", base, err)
			}
			// Best-effort: drop the owner directory if it became empty.
			// os.Remove fails on non-empty dirs, which is exactly what
			// we want here (other teams under the same owner survive).
			if owner != "" {
				_ = os.Remove(filepath.Join(workspaceDir(), owner))
			}

			return printJSON(map[string]any{
				"status":       "removed",
				"removed":      base,
				"removed_runs": removedRuns,
			})
		},
	}
}
