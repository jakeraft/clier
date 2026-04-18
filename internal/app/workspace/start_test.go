package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	"github.com/jakeraft/clier/internal/domain"
)

func TestCollectRunnableAgents_RejectsUnknownAgentType(t *testing.T) {
	t.Parallel()

	state := &Manifest{
		Owner: "org",
		Name:  "root",
		Teams: []StoredTeamState{{
			Owner: "org",
			Name:  "root",
			Projection: TeamProjection{
				Name:      "root",
				AgentType: "weird-agent",
			},
		}},
	}

	_, err := CollectRunnableAgents(state)
	var fault *domain.Fault
	if !errors.As(err, &fault) || fault.Kind != domain.KindUnsupportedKind {
		t.Fatalf("expected unsupported kind fault, got %v", err)
	}
}

func TestMarkFirstRun_MarksOnce(t *testing.T) {
	t.Parallel()

	manifest := &Manifest{}
	now := time.Date(2026, 4, 18, 1, 2, 3, 0, time.UTC)

	hint := MarkFirstRun(manifest, "run-123", func() time.Time { return now })
	if hint == "" {
		t.Fatal("expected hint on first mark")
	}
	if manifest.FirstRunAt == nil || !manifest.FirstRunAt.Equal(now) {
		t.Fatalf("FirstRunAt = %v, want %v", manifest.FirstRunAt, now)
	}

	hint = MarkFirstRun(manifest, "run-123", time.Now)
	if hint != "" {
		t.Fatalf("expected empty hint on second mark, got %q", hint)
	}
}

func TestValidateWorkingCopy_AcceptsAbstractManagerRoot(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	agentBase := filepath.Join(base, filepath.FromSlash(AgentWorkspaceLocalPath("org", "coder")))
	required := []string{
		filepath.Join(agentBase, "AGENTS.md"),
		filepath.Join(agentBase, ".clier", "work-log-protocol.md"),
		filepath.Join(agentBase, ".clier", TeamProtocolFileName()),
	}
	for _, path := range required {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	manifest := &Manifest{
		Kind:  string(api.KindTeam),
		Owner: "org",
		Name:  "root",
		Teams: []StoredTeamState{
			{
				Owner: "org",
				Name:  "root",
				Projection: TeamProjection{
					Name:      "root",
					AgentType: "manager",
					Children: []ChildProjection{{
						Owner: "org",
						Name:  "coder",
					}},
				},
			},
			{
				Owner:    "org",
				Name:     "coder",
				LocalDir: AgentWorkspaceLocalPath("org", "coder"),
				Projection: TeamProjection{
					Name:      "coder",
					AgentType: "codex",
					Command:   "codex",
				},
			},
		},
	}

	if err := ValidateWorkingCopy(base, manifest, fs, nil); err != nil {
		t.Fatalf("ValidateWorkingCopy: %v", err)
	}
}

func TestValidateWorkingCopy_RejectsUnknownAgentType(t *testing.T) {
	t.Parallel()

	manifest := &Manifest{
		Kind:  string(api.KindTeam),
		Owner: "org",
		Name:  "root",
		Teams: []StoredTeamState{{
			Owner: "org",
			Name:  "root",
			Projection: TeamProjection{
				Name:      "root",
				AgentType: "unknown",
			},
		}},
	}

	err := ValidateWorkingCopy(t.TempDir(), manifest, filesystem.New(), nil)
	var fault *domain.Fault
	if !errors.As(err, &fault) || fault.Kind != domain.KindUnsupportedKind {
		t.Fatalf("expected unsupported kind error, got %v", err)
	}
}
