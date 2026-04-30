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
var ErrRunNotFound = errors.New("run not found")

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
	StoppedAt   *time.Time `json:"stopped_at,omitempty"`
	Messages    []Message  `json:"messages,omitempty"`
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
	From      *string   `json:"from,omitempty"`
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

// Save writes run.json under the plan's RunDir, creating it if missing.
func (s *Store) Save(plan *Plan) error {
	if err := os.MkdirAll(plan.RunDir, 0o755); err != nil {
		return fmt.Errorf("create run dir: %w", err)
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	return os.WriteFile(filepath.Join(plan.RunDir, planFileName), data, 0o644)
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

// PurgeRunArtifacts removes every per-agent git clone destination plus
// any protocol files, leaving run.json behind so `clier run list` and
// `clier run view` keep working after stop.
func (s *Store) PurgeRunArtifacts(plan *Plan) error {
	for _, agent := range plan.Agents {
		if agent.GitDest != "" {
			if err := os.RemoveAll(filepath.Join(plan.RunDir, filepath.FromSlash(agent.GitDest))); err != nil {
				return fmt.Errorf("purge git clone for %s: %w", agent.ID, err)
			}
		}
		if agent.ProtocolDest != "" {
			if err := os.RemoveAll(filepath.Join(plan.RunDir, filepath.FromSlash(agent.ProtocolDest))); err != nil {
				return fmt.Errorf("purge protocol for %s: %w", agent.ID, err)
			}
		}
	}
	// Remove the protocols/ subdir if it ended up empty after the per-file
	// removals — leaves a tidy `<run_dir>/{run.json}` for retrospection.
	_ = os.Remove(filepath.Join(plan.RunDir, "protocols"))
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
	return &plan, nil
}
