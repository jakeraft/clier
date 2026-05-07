package runplan

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	StatusRunning = "running"
	StatusStopped = "stopped"

	planFileName = "run.json"
)

// ErrRunNotFound is returned when no run plan exists for the given runID.
var ErrRunNotFound = errors.New("run not found in ~/.clier/runs")

// ErrRunStopped is returned by tell/attach when the addressed run has
// been stopped (status flipped + run dir purged). Surfaces a clean
// rejection instead of a generic "not found" so the caller knows the
// run existed but is no longer live.
var ErrRunStopped = errors.New("run has been stopped")

// Plan is everything `clier run` needs to drive a started session — what
// was cloned, where it lives, and what's been said to whom. Mirrors the
// agent-grouped RunManifest (ADR-0002 §2) so post-stop introspection
// (`clier run view`) can show the same shape that mint emitted.
type Plan struct {
	RunID       string     `json:"run_id"`
	SessionName string     `json:"session_name"`
	RunDir      string     `json:"run_dir"`
	Namespace   string     `json:"namespace"`
	TeamName    string     `json:"team_name"`
	Agents      []Agent    `json:"agents"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	StoppedAt   *time.Time `json:"stopped_at"`
	Messages    []Message  `json:"messages"`
}

// Agent is a single tmux window's spec. The fs-resolved AbsCwd is baked
// in so tell/attach/stop can run without re-deriving it; the source
// fields (GitRepo*, ProtocolDest) are kept for retrospection.
type Agent struct {
	ID           string   `json:"id"`
	Window       int      `json:"window"`
	AbsCwd       string   `json:"abs_cwd"`
	GitRepoURL   string   `json:"git_repo_url"`
	GitSubpath   string   `json:"git_subpath"`
	GitDest      string   `json:"git_dest"`
	ProtocolDest string   `json:"protocol_dest"`
	Command      string   `json:"command"`
	Args         []string `json:"args"`
	AgentType    string   `json:"agent_type"`
}

// Message records a tell delivery — useful for `clier run view` audit.
type Message struct {
	From      *string   `json:"from"`
	To        string    `json:"to"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// FindAgent returns the agent with the given ID, if any.
func (p *Plan) FindAgent(agentID string) (*Agent, bool) {
	for i := range p.Agents {
		if p.Agents[i].ID == agentID {
			return &p.Agents[i], true
		}
	}
	return nil, false
}

// MarkStopped flips Status to stopped and stamps StoppedAt.
func (p *Plan) MarkStopped() {
	now := time.Now()
	p.Status = StatusStopped
	p.StoppedAt = &now
}

// AppendMessage records a delivered tell.
func (p *Plan) AppendMessage(from *string, to, content string) {
	var fromCopy *string
	if from != nil {
		v := *from
		fromCopy = &v
	}
	p.Messages = append(p.Messages, Message{
		From:      fromCopy,
		To:        to,
		Content:   content,
		CreatedAt: time.Now(),
	})
}

// Store persists run plans under a runs root (default ~/.clier/runs).
type Store struct {
	root string
}

func NewStore(rootDir string) *Store {
	return &Store{root: rootDir}
}

// RunDir returns the per-run directory path. Each agent's git.dest and
// protocol.dest are relative to this path (ADR-0002 §6).
func (s *Store) RunDir(runID string) string {
	return filepath.Join(s.root, runID)
}

// Create writes run.json for a fresh run, creating the run dir. Use
// once per `clier run start`.
func (s *Store) Create(plan *Plan) error {
	if err := os.MkdirAll(plan.RunDir, 0o755); err != nil {
		return fmt.Errorf("create run dir: %w", err)
	}
	return s.write(plan)
}

// Update rewrites run.json for an existing run — refusing if the run
// dir is missing. This is the ordering hinge that closes the
// stop ↔ tell race: once Stop runs PurgeRun, an in-flight Tell that
// reaches Update finds the dir gone and returns ErrRunStopped instead
// of MkdirAll-resurrecting a half-stopped layout. Tell, AppendMessage,
// MarkStopped+Save all go through Update.
func (s *Store) Update(plan *Plan) error {
	if _, err := os.Stat(plan.RunDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrRunStopped
		}
		return fmt.Errorf("stat run dir: %w", err)
	}
	return s.write(plan)
}

// write performs the actual marshal+write. 0o600 — the body of run.json
// can carry agent protocol templates and message history; restrict to
// the user that owns the run dir.
func (s *Store) write(plan *Plan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	return os.WriteFile(filepath.Join(plan.RunDir, planFileName), data, 0o600)
}

// Load reads run.json for the given runID. ErrRunNotFound when absent.
func (s *Store) Load(runID string) (*Plan, error) {
	return loadPlan(filepath.Join(s.RunDir(runID), planFileName))
}

// List returns every plan under the runs root, sorted newest-first.
func (s *Store) List() ([]*Plan, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read runs dir: %w", err)
	}
	plans := make([]*Plan, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		plan, err := loadPlan(filepath.Join(s.root, entry.Name(), planFileName))
		if err != nil {
			if errors.Is(err, ErrRunNotFound) {
				continue
			}
			return nil, fmt.Errorf("load run %s: %w", entry.Name(), err)
		}
		plans = append(plans, plan)
	}
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].StartedAt.After(plans[j].StartedAt)
	})
	return plans, nil
}

// PurgeRun removes the entire run directory — every per-agent clone,
// the protocols/ subdir, and run.json itself. Stop is final: a stopped
// run leaves no trace in `~/.clier/runs/`, so `clier run list` only
// surfaces live runs and the per-run prepare layout never accumulates.
func (s *Store) PurgeRun(plan *Plan) error {
	if plan.RunDir == "" {
		return nil
	}
	if err := os.RemoveAll(plan.RunDir); err != nil {
		return fmt.Errorf("purge run dir %s: %w", plan.RunDir, err)
	}
	return nil
}

func loadPlan(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrRunNotFound
		}
		return nil, fmt.Errorf("read run plan: %w", err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("unmarshal run plan %s: %w", path, err)
	}
	// Empty array surfaces as `[]` everywhere — never nil/null/missing.
	// json.Unmarshal of a missing/null Messages field yields nil; the
	// normalize keeps consumers from having to branch on the encoding.
	if plan.Messages == nil {
		plan.Messages = []Message{}
	}
	return &plan, nil
}
