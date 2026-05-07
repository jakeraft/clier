package runner

import (
	"context"
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

// safeJoinUnderRunDir joins a server-supplied relative path onto the
// run dir and rejects anything that escapes (`..`, absolute paths,
// NUL bytes, resolved location outside runDir, empty runDir). The
// server is the trusted source of paths in v1, but defense-in-depth
// keeps a compromised / spoofed server response from writing outside
// `~/.clier/runs/<run_id>/`.
func safeJoinUnderRunDir(runDir, rel string) (string, error) {
	if runDir == "" {
		// A blank base would let `filepath.Clean(rel)` resolve to a
		// path the prefix check is no longer evaluating — the helper
		// has no safe answer for "anchor anywhere".
		return "", fmt.Errorf("runDir must not be empty")
	}
	if strings.ContainsRune(rel, 0) {
		// NUL truncates strings on most C-backed filesystems and on
		// some Go syscalls. `filepath.Clean` does not strip it, so a
		// `"safe\x00/../etc/passwd"` value would survive cleaning and
		// be opened as `safe` while actually pointing at `/etc/passwd`
		// downstream.
		return "", fmt.Errorf("path contains NUL byte: %q", rel)
	}
	if rel == "" {
		return filepath.Clean(runDir), nil
	}
	cleanRel := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(cleanRel) {
		return "", fmt.Errorf("absolute path rejected: %q", rel)
	}
	// filepath.Clean leaves leading "../" intact, which lets us catch
	// escapes by checking the cleaned join against the runDir prefix.
	joined := filepath.Clean(filepath.Join(runDir, cleanRel))
	cleanRunDir := filepath.Clean(runDir)
	if joined != cleanRunDir &&
		!strings.HasPrefix(joined, cleanRunDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes run dir: %q", rel)
	}
	return joined, nil
}

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
		// Surface the server's "<status> <title>: <detail>" line verbatim
		// so the CLI's error contract stays uniform across every command.
		return nil, err
	}
	if manifest.RunID == "" {
		return nil, errors.New("server returned empty run_id")
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
		// Roll back partial tmux + filesystem state so a retry starts
		// clean. Rollback failures are best-effort but loud — a wedged
		// tmux session or undeletable run dir would otherwise leave
		// debris that the next `run start` attempt cannot clear.
		if err := r.tmux.KillSession(sessionName); err != nil {
			fmt.Fprintf(os.Stderr, "warning: rollback kill-session failed for %s: %s\n", sessionName, err)
		}
		if err := os.RemoveAll(runDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: rollback remove run dir failed for %s: %s\n", runDir, err)
		}
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
		protoPath, err := safeJoinUnderRunDir(runDir, spec.Prepare.Protocol.Dest)
		if err != nil {
			return nil, fmt.Errorf("protocol path for %s: %w", spec.ID, err)
		}
		if err := os.MkdirAll(filepath.Dir(protoPath), 0o755); err != nil {
			return nil, fmt.Errorf("create protocol dir for %s: %w", spec.ID, err)
		}
		// 0o600 — the protocol body and any embedded message history
		// can leak agent context on a shared host; restrict to owner.
		if err := os.WriteFile(protoPath, []byte(spec.Prepare.Protocol.Content), 0o600); err != nil {
			return nil, fmt.Errorf("write protocol for %s: %w", spec.ID, err)
		}
	}
	for _, spec := range manifest.Agents {
		dest, err := safeJoinUnderRunDir(runDir, spec.Prepare.Git.Dest)
		if err != nil {
			return nil, fmt.Errorf("git dest for %s: %w", spec.ID, err)
		}
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
		Messages:    []runplan.Message{}, // empty surfaces as [], never nil
	}

	// 2. Open tmux session and one window per agent. agent's cwd =
	//    git.dest [+ "/" + git.subpath] joined onto runDir.
	for i, spec := range manifest.Agents {
		absCwd, err := safeJoinUnderRunDir(runDir, spec.Prepare.Git.Dest)
		if err != nil {
			return nil, fmt.Errorf("cwd path for %s: %w", spec.ID, err)
		}
		if spec.Prepare.Git.Subpath != "" {
			absCwd, err = safeJoinUnderRunDir(runDir,
				filepath.Join(filepath.FromSlash(spec.Prepare.Git.Dest),
					filepath.FromSlash(spec.Prepare.Git.Subpath)))
			if err != nil {
				return nil, fmt.Errorf("subpath cwd for %s: %w", spec.ID, err)
			}
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
		// startup (cold node/npm boot routinely exceeds 2s on a fresh
		// machine boot).
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

	if err := r.store.Create(plan); err != nil {
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
	if plan.Status == runplan.StatusStopped {
		return runplan.ErrRunStopped
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
	if fromAgent != nil && *fromAgent != "" {
		if _, ok := plan.FindAgent(*fromAgent); !ok {
			return fmt.Errorf("%w: %s", ErrAgentNotInRun, *fromAgent)
		}
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
	return r.store.Update(plan)
}

// Stop kills the tmux session and frees the disk used by per-agent
// clones, protocol files, and run.json itself. The flow has a single
// race-closing hinge: status flips to "stopped" + Update *first*, so an
// in-flight Tell that reaches Update concurrent with the PurgeRun
// either sees the stopped status (Load before flip) and returns
// ErrRunStopped, or finds the run dir gone (Load after flip but
// Update after purge) and bubbles ErrRunStopped from Update. Either
// way the partial Save can no longer resurrect a half-stopped layout.
func (r *Runner) Stop(runID string) error {
	plan, err := r.store.Load(runID)
	if err != nil {
		return err
	}
	if plan.Status == runplan.StatusStopped {
		// Idempotent — purge any leftover dir and return success.
		return r.store.PurgeRun(plan)
	}
	plan.MarkStopped()
	if err := r.store.Update(plan); err != nil {
		return fmt.Errorf("flip run to stopped: %w", err)
	}
	r.gracefulExit(plan)
	if err := r.tmux.KillSession(plan.SessionName); err != nil {
		return fmt.Errorf("kill tmux session: %w", err)
	}
	return r.store.PurgeRun(plan)
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
	// context.WithTimeout is monotonic-clock backed and cancellable, so
	// laptop sleep / NTP jumps don't break the deadline (a `time.Now()`
	// loop trips on monotonic stalls during system suspend), and a
	// caller Ctrl-C cancels the wait without leaving an orphan poll.
	ctx, cancel := context.WithTimeout(context.Background(), readyDeadline)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		title, err := r.tmux.PaneTitle(sessionName, agent.Window)
		if err != nil {
			return fmt.Errorf("inspect pane title for %s: %w", agent.ID, err)
		}
		if strings.Contains(title, profile.readyMarker) {
			return nil
		}
		select {
		case <-ctx.Done():
			return &ErrReadyTimeout{AgentID: agent.ID, Timeout: readyDeadline}
		case <-ticker.C:
		}
	}
}

// gracefulExit sends each vendor's exit keyword (e.g. claude `/exit`) so
// the agent can flush state before tmux kills the session. Send failures
// are logged to stderr but do not abort Stop — the kill-session that
// follows is unconditional, and a wedged pane is exactly the case
// gracefulExit cannot handle anyway. Surfacing the warning lets the
// operator see when graceful drainage was bypassed.
func (r *Runner) gracefulExit(plan *runplan.Plan) {
	for _, agent := range plan.Agents {
		profile := profileFor(agent.AgentType)
		if profile.exitCommand == "" {
			continue
		}
		if err := r.tmux.SendLine(plan.SessionName, agent.Window, profile.exitCommand); err != nil {
			fmt.Fprintf(os.Stderr, "warning: graceful exit failed for %s: %s\n", agent.ID, err)
		}
	}
}
