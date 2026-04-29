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

	planFileName  = "run.json"
	mountsDirName = "mounts"
)

// ErrRunNotFound is returned when no run plan exists for the given runID.
var ErrRunNotFound = errors.New("run not found")

// Plan is everything `clier run` needs to drive a started session — what was
// cloned, where it lives, and what's been said to whom.
type Plan struct {
	RunID       string     `json:"run_id"`
	SessionName string     `json:"session_name"`
	RunDir      string     `json:"run_dir"`
	Namespace   string     `json:"namespace"`
	TeamName    string     `json:"team_name"`
	Mounts      []Mount    `json:"mounts"`
	Agents      []Agent    `json:"agents"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	StoppedAt   *time.Time `json:"stopped_at,omitempty"`
	Messages    []Message  `json:"messages,omitempty"`
}

// Mount is a clone destination on disk (resolved from RunManifest.Mount +
// the run's scratch dir).
type Mount struct {
	Name       string `json:"name"`
	GitRepoURL string `json:"git_repo_url"`
	GitSubpath string `json:"git_subpath"`
	LocalDir   string `json:"local_dir"`
}

// Agent is a single tmux window's spec, with the scratch-dir-resolved cwd
// pre-baked so tell/attach/stop can run without re-deriving it.
type Agent struct {
	ID        string   `json:"id"`
	Window    int      `json:"window"`
	Mount     string   `json:"mount"`
	Cwd       string   `json:"cwd"`
	AbsCwd    string   `json:"abs_cwd"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	AgentType string   `json:"agent_type"`
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

// RunDir returns the per-run directory path (parent of run.json + mounts/).
func (s *Store) RunDir(runID string) string {
	return filepath.Join(s.root, runID)
}

// MountsDir returns the absolute clones-base directory for a run.
func (s *Store) MountsDir(runID string) string {
	return filepath.Join(s.RunDir(runID), mountsDirName)
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

// PurgeMounts removes the cloned mounts for a run while leaving run.json
// behind, so `clier run list` and `clier run view` keep working after stop.
func (s *Store) PurgeMounts(runID string) error {
	mounts := s.MountsDir(runID)
	if err := os.RemoveAll(mounts); err != nil {
		return fmt.Errorf("purge mounts: %w", err)
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
	return &plan, nil
}
