package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/adapter/workspace"
	"github.com/jakeraft/clier/internal/app/session"
	"github.com/spf13/cobra"
)

const (
	tutorialImportURL    = "https://raw.githubusercontent.com/jakeraft/clier/main/tutorials/story-team"
	tutorialTeamID       = "ebfc4588-b1a9-45a6-a725-457eb4bbe875"       // Source: tutorials/story-team/15-team.json
	tutorialRootMemberID = "ebfc4588-aa01-4000-8000-000000000001"       // Source: tutorials/story-team/15-team.json
	tutorialMessage      = "Write a short mystery story"
)

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tutorial",
		Short: "Learn the clier workflow with an example team",
		Long: `Learn the clier workflow with an example team.

This tutorial uses the "story-team" — a hierarchical team of AI agents
that collaborates to write a short mystery story:

  chief-editor
  ├── section-editor-1
  │   ├── writer-1
  │   └── writer-2
  └── section-editor-2
      ├── writer-3
      └── writer-4

Run "clier tutorial start" to execute the following commands in sequence:

  1. clier import ` + tutorialImportURL + `
  2. clier session start ` + tutorialTeamID + `
  3. clier session tell --session <session-id> --to ` + tutorialRootMemberID + ` "` + tutorialMessage + `"

After the session starts, check progress with:

  clier session logs <session-id>
  # clier session attach <session-id>  (coming soon)`,
	}
	cmd.AddCommand(newTutorialStartCmd())
	return cmd
}

func newTutorialStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "start",
		Short:       "Run the tutorial",
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			ctx := cmd.Context()

			// Step 1: Import story-team.
			fmt.Fprintln(os.Stderr, "Step 1/3: Importing story-team...")
			src := strings.TrimRight(tutorialImportURL, "/") + "/index.json"
			data, err := readSource(src)
			if err != nil {
				return fmt.Errorf("fetch tutorial: %w", err)
			}

			var idx indexFile
			if err := json.Unmarshal(data, &idx); err != nil {
				return fmt.Errorf("parse index.json: %w", err)
			}
			base := basePath(src)
			for _, f := range idx.Files {
				fileSrc := joinPath(base, f)
				fileData, err := readSource(fileSrc)
				if err != nil {
					return fmt.Errorf("read %s: %w", fileSrc, err)
				}
				if err := importEnvelope(ctx, store, fileData); err != nil {
					return fmt.Errorf("import %s: %w", f, err)
				}
			}

			// Step 2: Start session.
			fmt.Fprintln(os.Stderr, "Step 2/3: Starting session...")
			t, err := store.GetTeam(ctx, tutorialTeamID)
			if err != nil {
				return fmt.Errorf("get team: %w", err)
			}

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := session.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			s, err := svc.Start(ctx, t, cfg.Auth)
			if err != nil {
				return fmt.Errorf("start session: %w", err)
			}

			// Step 3: Tell root member to start.
			fmt.Fprintln(os.Stderr, "Step 3/3: Telling chief-editor to start...")
			if err := svc.Send(ctx, s.ID, "", tutorialRootMemberID, tutorialMessage); err != nil {
				return fmt.Errorf("tell message: %w", err)
			}

			fmt.Fprintln(os.Stderr, "\nTutorial session started successfully.")
			fmt.Fprintln(os.Stderr, "\nNext steps:")
			fmt.Fprintf(os.Stderr, "  clier session logs %s\n", s.ID)
			fmt.Fprintf(os.Stderr, "  # clier session attach %s  (coming soon)\n", s.ID)

			return printJSON(s)
		},
	}
}
