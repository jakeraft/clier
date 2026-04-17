package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
)

func TestMaterializeAgent_WritesSkillsUnderOwnerAndName(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	writer := NewWriter(filesystem.New(), nil, map[string]*api.ResolvedResource{
		"alice/reviewer": {
			OwnerName: "alice",
			Name:      "reviewer",
			Snapshot:  []byte(`{"content":"# reviewer skill"}`),
		},
	})

	err := writer.MaterializeAgent(base, &TeamProjection{
		Name:      "coder",
		AgentType: "claude",
		Skills: []ResourceRefProjection{{
			Owner: "alice",
			Name:  "reviewer",
		}},
	}, "jakeraft/coder")
	if err != nil {
		t.Fatalf("MaterializeAgent: %v", err)
	}

	skillPath := filepath.Join(base, ".claude", "skills", "alice", "reviewer", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", skillPath, err)
	}
	if string(data) != "# reviewer skill" {
		t.Fatalf("skill content = %q", string(data))
	}
}
