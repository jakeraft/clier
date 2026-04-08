# Tutorial Command & Root Help Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `clier tutorial` command with `start` subcommand and improve root help to explain the clier workflow.

**Architecture:** Two file changes — modify `cmd/root.go` to add a `Long` description, create `cmd/tutorial.go` with a parent command (help guide in `Long`) and a `start` subcommand that reuses service-layer functions from the same `cmd` package to import story-team, start a session, and send a hardcoded message.

**Tech Stack:** Go, Cobra CLI framework, existing clier services (import, session, terminal, workspace)

---

### Task 1: Improve root help text

**Files:**
- Modify: `cmd/root.go:24-29` (rootCmd definition)

- [ ] **Step 1: Add `Long` field to rootCmd**

In `cmd/root.go`, add a `Long` field to the `rootCmd` variable:

```go
var rootCmd = &cobra.Command{
	Use:   "clier",
	Short: "Orchestrate AI coding agent teams in isolated workspaces",
	Long: `Orchestrate AI coding agent teams in isolated workspaces.

Building blocks (profile, prompt, env, repo) define agent capabilities.
Combine them into a member, assemble members into a team with
leader-worker relations, then start a session to launch the agents.
Monitor progress through messages and logs, or open the dashboard.

New to clier? Run "clier tutorial" for a step-by-step guide.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
```

- [ ] **Step 2: Verify help output**

Run: `go run . --help`

Expected: The output now shows the full `Long` text above the `Available Commands` section.

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat: improve root help with building-block workflow overview

Refs: #29"
```

---

### Task 2: Create tutorial parent command with help guide

**Files:**
- Create: `cmd/tutorial.go`

- [ ] **Step 1: Create `cmd/tutorial.go` with parent command and init**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

const tutorialImportURL = "https://raw.githubusercontent.com/jakeraft/clier/main/tutorials/story-team"
const tutorialTeamID = "ebfc4588-b1a9-45a6-a725-457eb4bbe875"
const tutorialRootMemberID = "ebfc4588-aa01-4000-8000-000000000001"
const tutorialMessage = "Write a short mystery story"

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
  3. clier session send --session <session-id> --to ` + tutorialRootMemberID + ` "` + tutorialMessage + `"

After the session starts, check progress with:

  clier session logs <session-id>
  # clier session attach <session-id>  (coming soon)`,
	}
	cmd.AddCommand(newTutorialStartCmd())
	return cmd
}
```

- [ ] **Step 2: Verify tutorial help output**

Run: `go run . tutorial`

Expected: Shows the Long text with the story-team hierarchy, the 3 commands, and the next-steps section.

Run: `go run . tutorial help`

Expected: Same output.

- [ ] **Step 3: Commit**

```bash
git add cmd/tutorial.go
git commit -m "feat: add tutorial command with step-by-step help guide

Refs: #29"
```

---

### Task 3: Implement `tutorial start` subcommand

**Files:**
- Modify: `cmd/tutorial.go` (add `newTutorialStartCmd` function)

- [ ] **Step 1: Add the `newTutorialStartCmd` function to `cmd/tutorial.go`**

Add the following after `newTutorialCmd`:

```go
func newTutorialStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "start",
		Short:       "Run the tutorial (import story-team, start session, send message)",
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

			// Step 3: Send message to root member.
			fmt.Fprintln(os.Stderr, "Step 3/3: Sending message to chief-editor...")
			if err := svc.Send(ctx, s.ID, "", tutorialRootMemberID, tutorialMessage); err != nil {
				return fmt.Errorf("send message: %w", err)
			}

			fmt.Fprintln(os.Stderr, "\nTutorial session started successfully.")
			fmt.Fprintln(os.Stderr, "\nNext steps:")
			fmt.Fprintf(os.Stderr, "  clier session logs %s\n", s.ID)
			fmt.Fprintf(os.Stderr, "  # clier session attach %s  (coming soon)\n", s.ID)

			return printJSON(s)
		},
	}
}
```

- [ ] **Step 2: Add missing imports to `cmd/tutorial.go`**

Update the import block at the top of `cmd/tutorial.go`:

```go
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
```

- [ ] **Step 3: Verify compilation**

Run: `go build .`

Expected: Builds successfully with no errors.

- [ ] **Step 4: Verify tutorial help still works**

Run: `go run . tutorial`

Expected: Shows the step-by-step help guide (unchanged from Task 2).

- [ ] **Step 5: Commit**

```bash
git add cmd/tutorial.go
git commit -m "feat: implement tutorial start command

Executes the full tutorial workflow:
import story-team, start session, send hardcoded message.

Refs: #29"
```

---

### Task 4: Verify agent-mode exclusion and final integration

**Files:**
- Read: `cmd/root.go:86-115` (filterAgentCommands)

- [ ] **Step 1: Verify tutorial is excluded from agent mode**

Check `cmd/root.go` `filterAgentCommands()`. The allowed map is `map[string]bool{"session": true}`. Since `tutorial` is not in this map, it is already excluded. No code change needed.

- [ ] **Step 2: Run full verification**

Run: `go build . && ./clier --help`

Expected: Root help shows the Long description with building-block overview.

Run: `./clier tutorial`

Expected: Shows the step-by-step tutorial guide with story-team hierarchy.

Run: `./clier tutorial help`

Expected: Same as above.

Run: `./clier tutorial start --help`

Expected: Shows start subcommand help with Short description.

- [ ] **Step 3: Run existing tests to ensure no regressions**

Run: `go test ./...`

Expected: All existing tests pass.

- [ ] **Step 4: Final commit (if any fixes needed)**

Only if previous steps required fixes. Otherwise skip.
