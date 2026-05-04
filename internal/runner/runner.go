package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/api"
	"github.com/jakeraft/clier/internal/git"
	"github.com/jakeraft/clier/internal/runplan"
	"github.com/jakeraft/clier/internal/tmux"
)

// ErrAgentNotInRun signals a tell/attach targeted an agent that the run
// does not contain. Surfaced verbatim — no message catalog.
var ErrAgentNotInRun = errors.New("agent not found in run")

// ErrReadyTimeout signals an agent's TUI did not surface its ready marker
// before the deadline.
type ErrReadyTimeout struct {
	AgentID string
	Timeout time.Duration
}

func (e *ErrReadyTimeout) Error() string {
	return fmt.Sprintf("agent %s did not become ready within %s", e.AgentID, e.Timeout)
}

// readyDeadline is the per-agent TUI ready timeout. Claude Code typically
// boots in <5s on a warm cache, <20s cold.
const readyDeadline = 60 * time.Second

// RunsAPI is the narrow port the runner uses on the api package — only
// MintRun is needed. Defined here so tests can swap in a fake.
type RunsAPI interface {
	MintRun(namespace, name string) (*api.RunManifest, error)
}

// Deps wires the runner with its collaborators. All four are required.
type Deps struct {
	API   RunsAPI
	Git   git.Git
	Tmux  tmux.Tmux
	Store *runplan.Store
}

// Runner orchestrates the thin tmux flow: mint → write protocols → clone
// → tmux launch → persist → tell/attach/stop.
type Runner struct {
	api   RunsAPI
	git   git.Git
	tmux  tmux.Tmux
	store *runplan.Store
	now   func() time.Time
}

func New(d Deps) *Runner {
	return &Runner{
		api:   d.API,
		git:   d.Git,
		tmux:  d.Tmux,
		store: d.Store,
		now:   time.Now,
	}
}

// Start mints a fresh RunManifest from the server, drops each agent's
// rendered protocol on disk, clones each agent's git source, and
// launches one tmux window per agent (ADR-0002 §7). Failures at any step
// roll back tmux state and remove the run scratch dir so a retry leaves
// no debris.
//
// RUN_ID is server-minted (ADR-0002 §3) — the CLI does not generate it.
// `run.args` from the manifest is sent verbatim after per-item
// shell-escape; the CLI does not substitute or rewrite tokens.
func (r *Runner) Start(namespace, name string) (*runplan.Plan, error) {
	manifest, err := r.api.MintRun(namespace, name)
	if err != nil {
		return nil, fmt.Errorf("mint run for %s/%s: %w", namespace, name, err)
	}
	if manifest.RunID == "" {
		return nil, fmt.Errorf("server returned empty run_id")
	}
	if len(manifest.Agents) == 0 {
		return nil, fmt.Errorf("team %s/%s has no runnable agents", namespace, name)
	}

	runID := manifest.RunID
	sessionName := "clier-" + runID
	runDir := r.store.RunDir(runID)

	success := false
	defer func() {
		if success {
			return
		}
		// Roll back partial tmux + filesystem state so a retry starts clean.
		_ = r.tmux.KillSession(sessionName)
		_ = os.RemoveAll(runDir)
	}()

	// 1. Materialize per-agent prepare items (ADR-0002 §6 layout). Order:
	//    write protocol files first, then clone — so a clone failure can
	//    short-circuit without leaving inconsistent on-disk state.
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create run dir: %w", err)
	}
	for _, spec := range manifest.Agents {
		// prepare.protocol is omitted for inline-vendor agents (e.g. codex
		// receives the protocol verbatim in run.args). Skip the write so
		// no redundant file lands on disk for them.
		if spec.Prepare.Protocol == nil {
			continue
		}
		protoPath := filepath.Join(runDir, filepath.FromSlash(spec.Prepare.Protocol.Dest))
		if err := os.MkdirAll(filepath.Dir(protoPath), 0o755); err != nil {
			return nil, fmt.Errorf("create protocol dir for %s: %w", spec.ID, err)
		}
		if err := os.WriteFile(protoPath, []byte(spec.Prepare.Protocol.Content), 0o644); err != nil {
			return nil, fmt.Errorf("write protocol for %s: %w", spec.ID, err)
		}
	}
	for _, spec := range manifest.Agents {
		dest := filepath.Join(runDir, filepath.FromSlash(spec.Prepare.Git.Dest))
		if err := r.git.Clone(spec.Prepare.Git.RepoURL, dest); err != nil {
			return nil, fmt.Errorf("clone for %s: %w", spec.ID, err)
		}
	}

	plan := &runplan.Plan{
		RunID:       runID,
		SessionName: sessionName,
		RunDir:      runDir,
		Namespace:   namespace,
		TeamName:    name,
		Status:      runplan.StatusRunning,
		StartedAt:   r.now(),
	}

	// 2. Open tmux session and one window per agent. agent's cwd =
	//    git.dest [+ "/" + git.subpath] joined onto runDir.
	for i, spec := range manifest.Agents {
		absCwd := filepath.Join(runDir, filepath.FromSlash(spec.Prepare.Git.Dest))
		if spec.Prepare.Git.Subpath != "" {
			absCwd = filepath.Join(absCwd, filepath.FromSlash(spec.Prepare.Git.Subpath))
		}
		var windowIdx int
		var werr error
		if i == 0 {
			windowIdx, werr = r.tmux.NewSession(sessionName, spec.ID, absCwd)
		} else {
			windowIdx, werr = r.tmux.NewWindow(sessionName, spec.ID, absCwd)
		}
		if werr != nil {
			return nil, fmt.Errorf("create tmux window for %s: %w", spec.ID, werr)
		}
		var protocolDest string
		if spec.Prepare.Protocol != nil {
			protocolDest = spec.Prepare.Protocol.Dest
		}
		plan.Agents = append(plan.Agents, runplan.Agent{
			ID:           spec.ID,
			Window:       windowIdx,
			AbsCwd:       absCwd,
			GitRepoURL:   spec.Prepare.Git.RepoURL,
			GitSubpath:   spec.Prepare.Git.Subpath,
			GitDest:      spec.Prepare.Git.Dest,
			ProtocolDest: protocolDest,
			Command:      spec.Run.Command,
			Args:         append([]string{}, spec.Run.Args...),
			AgentType:    spec.Run.AgentType,
		})
	}

	// 3. Send the launch line for each agent. command + args verbatim,
	//    with only per-item shell-escape on args (ADR-0002 §9 send-keys).
	for _, agent := range plan.Agents {
		line := joinCommandLine(agent.Command, agent.Args)
		if err := r.tmux.SendLine(sessionName, agent.Window, line); err != nil {
			return nil, fmt.Errorf("launch %s: %w", agent.ID, err)
		}
	}

	// 4. Auto-dismiss any vendor trust prompt before the readiness poll.
	//    Codex 0.121+ blocks on "Do you trust this directory?" right
	//    after launch (issue #19426 — no native skip flag); without this
	//    every run halts until an operator attaches and presses 1+Enter.
	//    Claude has no trust gate so its profile leaves trustResponse
	//    blank.
	for _, agent := range plan.Agents {
		profile := profileFor(agent.AgentType)
		if profile.trustResponse == "" {
			continue
		}
		// Brief pause so the prompt has time to render before we send
		// the keystroke. Polling for the prompt text is theoretically
		// stronger but adds tmux capture-pane I/O for every codex
		// launch; a fixed 3s delay covers cold-cache codex 0.125
		// startup (cold node/npm boot routinely exceeds 1.5s on first
		// run after a system restart, so the earlier 1.5s budget would
		// race the prompt and silently leave the run blocked).
		time.Sleep(3 * time.Second)
		if err := r.tmux.SendLine(sessionName, agent.Window, profile.trustResponse); err != nil {
			return nil, fmt.Errorf("auto-trust %s: %w", agent.ID, err)
		}
	}

	for _, agent := range plan.Agents {
		if err := r.waitReady(sessionName, agent); err != nil {
			return nil, err
		}
	}

	if err := r.store.Save(plan); err != nil {
		return nil, fmt.Errorf("save run plan: %w", err)
	}
	success = true
	return plan, nil
}

// Tell sends a message to the target agent's tmux window and records it
// in the run plan. fromAgent is optional: when present, the message is
// prefixed with `[Message from <fromAgent>] ` so the recipient sees the
// origin in their TUI.
//
// Refuses delivery when the tmux session is gone — without this guard the
// send-keys lands in whatever shell prompt happens to occupy that window
// after the agent crashed.
func (r *Runner) Tell(runID string, fromAgent *string, toAgent string, content string) error {
	plan, err := r.store.Load(runID)
	if err != nil {
		return err
	}
	if plan.Status != runplan.StatusRunning {
		return fmt.Errorf("run %s is not active (status=%s)", runID, plan.Status)
	}
	alive, err := r.tmux.HasSession(plan.SessionName)
	if err != nil {
		return fmt.Errorf("inspect tmux session: %w", err)
	}
	if !alive {
		return &tmux.ErrSessionGone{Session: plan.SessionName}
	}
	agent, ok := plan.FindAgent(toAgent)
	if !ok {
		return fmt.Errorf("%w: %s", ErrAgentNotInRun, toAgent)
	}
	text := strings.TrimSpace(content)
	if text == "" {
		return errors.New("message content is empty")
	}
	delivery := text
	if fromAgent != nil && *fromAgent != "" {
		delivery = fmt.Sprintf("[Message from %s] %s", *fromAgent, text)
	}
	if err := r.tmux.SendLine(plan.SessionName, agent.Window, delivery); err != nil {
		return fmt.Errorf("deliver message: %w", err)
	}
	plan.AppendMessage(fromAgent, toAgent, text)
	return r.store.Save(plan)
}

// Stop kills the tmux session, marks the plan as stopped, and frees the
// disk used by per-agent clones and protocol files. The run plan json
// stays on disk so `clier run view` keeps working post-stop.
func (r *Runner) Stop(runID string) error {
	plan, err := r.store.Load(runID)
	if err != nil {
		return err
	}
	r.gracefulExit(plan)
	if err := r.tmux.KillSession(plan.SessionName); err != nil {
		return fmt.Errorf("kill tmux session: %w", err)
	}
	plan.MarkStopped()
	if err := r.store.Save(plan); err != nil {
		return fmt.Errorf("save stopped plan: %w", err)
	}
	return r.store.PurgeRunArtifacts(plan)
}

// Attach hands control of stdin/stdout/stderr to a tmux attach. When
// agentID is non-nil, the matching window is selected first.
func (r *Runner) Attach(runID string, agentID *string) error {
	plan, err := r.store.Load(runID)
	if err != nil {
		return err
	}
	var windowIdx *int
	if agentID != nil && *agentID != "" {
		agent, ok := plan.FindAgent(*agentID)
		if !ok {
			return fmt.Errorf("%w: %s", ErrAgentNotInRun, *agentID)
		}
		idx := agent.Window
		windowIdx = &idx
	}
	return r.tmux.Attach(plan.SessionName, windowIdx)
}

// List returns every persisted run, newest-first.
func (r *Runner) List() ([]*runplan.Plan, error) {
	return r.store.List()
}

// View returns a single run by id.
func (r *Runner) View(runID string) (*runplan.Plan, error) {
	return r.store.Load(runID)
}

func (r *Runner) waitReady(sessionName string, agent runplan.Agent) error {
	profile := profileFor(agent.AgentType)
	if profile.readyMarker == "" {
		return nil
	}
	deadline := time.Now().Add(readyDeadline)
	for time.Now().Before(deadline) {
		title, err := r.tmux.PaneTitle(sessionName, agent.Window)
		if err != nil {
			return fmt.Errorf("inspect pane title for %s: %w", agent.ID, err)
		}
		if strings.Contains(title, profile.readyMarker) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return &ErrReadyTimeout{AgentID: agent.ID, Timeout: readyDeadline}
}

func (r *Runner) gracefulExit(plan *runplan.Plan) {
	for _, agent := range plan.Agents {
		profile := profileFor(agent.AgentType)
		if profile.exitCommand == "" {
			continue
		}
		_ = r.tmux.SendLine(plan.SessionName, agent.Window, profile.exitCommand)
	}
}
