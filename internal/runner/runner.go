package runner

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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

// ResolverAPI is the narrow port the runner uses on the api package — only
// ResolveTeam is needed. Defined here so tests can swap in a fake.
type ResolverAPI interface {
	ResolveTeam(namespace, name string) (*api.RunManifest, error)
}

// Deps wires the runner with its collaborators. All four are required.
type Deps struct {
	API   ResolverAPI
	Git   git.Git
	Tmux  tmux.Tmux
	Store *runplan.Store
}

// Runner orchestrates the thin tmux flow: resolve → clone → tmux launch →
// persist → tell/attach/stop.
type Runner struct {
	api   ResolverAPI
	git   git.Git
	tmux  tmux.Tmux
	store *runplan.Store
	now   func() time.Time
	newID func() (string, error)
}

func New(d Deps) *Runner {
	return &Runner{
		api:   d.API,
		git:   d.Git,
		tmux:  d.Tmux,
		store: d.Store,
		now:   time.Now,
		newID: defaultRunID,
	}
}

// Start resolves the team into a RunManifest, clones each mount under the
// run scratch dir, launches one tmux window per agent, and persists the
// resulting plan.
func (r *Runner) Start(namespace, name string) (*runplan.Plan, error) {
	manifest, err := r.api.ResolveTeam(namespace, name)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s: %w", namespace, name, err)
	}
	if len(manifest.Agents) == 0 {
		return nil, fmt.Errorf("team %s/%s has no runnable agents", namespace, name)
	}

	runID, err := r.newID()
	if err != nil {
		return nil, fmt.Errorf("mint run id: %w", err)
	}
	sessionName := "clier-" + runID
	runDir := r.store.RunDir(runID)
	mountsDir := r.store.MountsDir(runID)

	planMounts := make([]runplan.Mount, 0, len(manifest.Mounts))
	for _, m := range manifest.Mounts {
		dest := filepath.Join(mountsDir, m.Name)
		if err := r.git.Clone(m.GitRepoURL, dest); err != nil {
			return nil, fmt.Errorf("clone mount %s: %w", m.Name, err)
		}
		planMounts = append(planMounts, runplan.Mount{
			Name:       m.Name,
			GitRepoURL: m.GitRepoURL,
			GitSubpath: m.GitSubpath,
			LocalDir:   dest,
		})
	}

	plan := &runplan.Plan{
		RunID:       runID,
		SessionName: sessionName,
		RunDir:      runDir,
		Namespace:   namespace,
		TeamName:    name,
		Mounts:      planMounts,
		Status:      runplan.StatusRunning,
		StartedAt:   r.now(),
	}

	for i, spec := range manifest.Agents {
		absCwd := filepath.Join(mountsDir, filepath.FromSlash(spec.Cwd))
		var windowIdx int
		var werr error
		if i == 0 {
			windowIdx, werr = r.tmux.NewSession(sessionName, spec.ID, absCwd)
		} else {
			windowIdx, werr = r.tmux.NewWindow(sessionName, spec.ID, absCwd)
		}
		if werr != nil {
			r.bestEffortKill(sessionName)
			return nil, fmt.Errorf("create tmux window for %s: %w", spec.ID, werr)
		}
		plan.Agents = append(plan.Agents, runplan.Agent{
			ID:        spec.ID,
			Window:    windowIdx,
			Mount:     spec.Mount,
			Cwd:       spec.Cwd,
			AbsCwd:    absCwd,
			Command:   spec.Command,
			Args:      append([]string{}, spec.Args...),
			AgentType: spec.AgentType,
		})
	}

	for _, agent := range plan.Agents {
		line := joinCommandLine(agent.Command, agent.Args)
		if err := r.tmux.SendLine(sessionName, agent.Window, line); err != nil {
			r.bestEffortKill(sessionName)
			return nil, fmt.Errorf("launch %s: %w", agent.ID, err)
		}
	}

	for _, agent := range plan.Agents {
		if err := r.waitReady(sessionName, agent); err != nil {
			r.bestEffortKill(sessionName)
			return nil, err
		}
	}

	if err := r.store.Save(plan); err != nil {
		r.bestEffortKill(sessionName)
		return nil, fmt.Errorf("save run plan: %w", err)
	}
	return plan, nil
}

// Tell sends a message to the target agent's tmux window and records it in
// the run plan. fromAgent is optional: when present, the message is
// prefixed with `[Message from <fromAgent>] ` so the recipient sees the
// origin in their TUI.
func (r *Runner) Tell(runID string, fromAgent *string, toAgent string, content string) error {
	plan, err := r.store.Load(runID)
	if err != nil {
		return err
	}
	if plan.Status != runplan.StatusRunning {
		return fmt.Errorf("run %s is not active (status=%s)", runID, plan.Status)
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
// disk used by cloned mounts. The run plan json stays on disk so `clier
// run view` keeps working post-stop.
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
	return r.store.PurgeMounts(runID)
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

func (r *Runner) bestEffortKill(session string) {
	_ = r.tmux.KillSession(session)
}

func defaultRunID() (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return timestamp + "-" + hex.EncodeToString(b[:]), nil
}
