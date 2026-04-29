package runner

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/api"
	"github.com/jakeraft/clier/internal/runplan"
)

type fakeAPI struct {
	manifest *api.RunManifest
}

func (f *fakeAPI) ResolveTeam(namespace, name string) (*api.RunManifest, error) {
	return f.manifest, nil
}

type fakeGit struct {
	clones []clone
}

type clone struct{ repoURL, dest string }

func (g *fakeGit) Clone(repoURL, dest string) error {
	g.clones = append(g.clones, clone{repoURL, dest})
	return nil
}

type tmuxOp struct {
	kind   string
	target string
	value  string
}

type fakeTmux struct {
	ops      []tmuxOp
	nextIdx  int
	titleFor func(session string, idx int) string
}

func (t *fakeTmux) NewSession(session, name, cwd string) (int, error) {
	idx := t.nextIdx
	t.nextIdx++
	t.ops = append(t.ops, tmuxOp{kind: "new-session", target: session, value: name + "@" + cwd})
	return idx, nil
}

func (t *fakeTmux) NewWindow(session, name, cwd string) (int, error) {
	idx := t.nextIdx
	t.nextIdx++
	t.ops = append(t.ops, tmuxOp{kind: "new-window", target: session, value: name + "@" + cwd})
	return idx, nil
}

func (t *fakeTmux) SendLine(session string, idx int, line string) error {
	t.ops = append(t.ops, tmuxOp{kind: "send", target: session, value: line})
	return nil
}

func (t *fakeTmux) Attach(session string, idx *int) error {
	t.ops = append(t.ops, tmuxOp{kind: "attach", target: session})
	return nil
}

func (t *fakeTmux) KillSession(session string) error {
	t.ops = append(t.ops, tmuxOp{kind: "kill", target: session})
	return nil
}

func (t *fakeTmux) PaneTitle(session string, idx int) (string, error) {
	if t.titleFor != nil {
		return t.titleFor(session, idx), nil
	}
	return "Claude", nil
}

func (t *fakeTmux) HasSession(session string) (bool, error) { return true, nil }

func TestStartHappyPath_singleAgent(t *testing.T) {
	manifest := &api.RunManifest{
		Mounts: []api.Mount{
			{
				Name:       "jakeraft.clier-qa-claude",
				GitRepoURL: "https://github.com/jakeraft/clier-qa",
				GitSubpath: "teams/clier-qa-claude",
			},
		},
		Agents: []api.AgentSpec{
			{
				ID:        "jakeraft.clier-qa-claude",
				Window:    0,
				Mount:     "jakeraft.clier-qa-claude",
				Cwd:       "jakeraft.clier-qa-claude/teams/clier-qa-claude",
				Command:   "CLIER_AGENT= claude --setting-sources project",
				Args:      []string{"--append-system-prompt", "# Team Protocol\n"},
				AgentType: "claude",
			},
		},
	}

	root := t.TempDir()
	store := runplan.NewStore(root)
	gitFake := &fakeGit{}
	tmuxFake := &fakeTmux{}
	r := New(Deps{API: &fakeAPI{manifest: manifest}, Git: gitFake, Tmux: tmuxFake, Store: store})
	r.newID = func() (string, error) { return "rid-1", nil }

	plan, err := r.Start("jakeraft", "clier-qa-claude")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if plan.RunID != "rid-1" {
		t.Errorf("RunID: got %q, want rid-1", plan.RunID)
	}
	if plan.SessionName != "clier-rid-1" {
		t.Errorf("SessionName: got %q", plan.SessionName)
	}

	wantCloneDest := filepath.Join(store.MountsDir("rid-1"), "jakeraft.clier-qa-claude")
	if len(gitFake.clones) != 1 || gitFake.clones[0].dest != wantCloneDest {
		t.Errorf("clones: got %+v, want one clone to %q", gitFake.clones, wantCloneDest)
	}

	gotKinds := []string{}
	for _, op := range tmuxFake.ops {
		gotKinds = append(gotKinds, op.kind)
	}
	wantKinds := []string{"new-session", "send"}
	if !equalSlice(gotKinds, wantKinds) {
		t.Errorf("tmux op order: got %v, want %v", gotKinds, wantKinds)
	}

	for _, op := range tmuxFake.ops {
		if op.kind == "send" {
			if !strings.Contains(op.value, "CLIER_AGENT= claude --setting-sources project") {
				t.Errorf("send-keys missing command verbatim: %q", op.value)
			}
			if !strings.Contains(op.value, "'# Team Protocol\n'") {
				t.Errorf("send-keys missing shell-escaped args: %q", op.value)
			}
		}
	}

	loaded, err := store.Load("rid-1")
	if err != nil {
		t.Fatalf("Load saved plan: %v", err)
	}
	if loaded.Status != runplan.StatusRunning {
		t.Errorf("Status: got %q, want running", loaded.Status)
	}
	if len(loaded.Agents) != 1 || loaded.Agents[0].AgentType != "claude" {
		t.Errorf("Agents: got %+v", loaded.Agents)
	}
}

func TestStartHappyPath_multiAgent(t *testing.T) {
	manifest := &api.RunManifest{
		Mounts: []api.Mount{
			{Name: "ns.team", GitRepoURL: "https://example/repo", GitSubpath: ""},
		},
		Agents: []api.AgentSpec{
			{ID: "ns.a", Window: 0, Mount: "ns.team", Cwd: "ns.team", Command: "claude", Args: []string{}, AgentType: "claude"},
			{ID: "ns.b", Window: 1, Mount: "ns.team", Cwd: "ns.team", Command: "claude", Args: []string{}, AgentType: "claude"},
		},
	}
	r := New(Deps{
		API:   &fakeAPI{manifest: manifest},
		Git:   &fakeGit{},
		Tmux:  &fakeTmux{},
		Store: runplan.NewStore(t.TempDir()),
	})
	r.newID = func() (string, error) { return "rid-2", nil }

	plan, err := r.Start("ns", "team")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(plan.Agents) != 2 {
		t.Fatalf("Agents: got %d, want 2", len(plan.Agents))
	}
	if plan.Agents[0].Window == plan.Agents[1].Window {
		t.Errorf("windows must differ, got %d/%d", plan.Agents[0].Window, plan.Agents[1].Window)
	}
}

func TestTellRecordsMessage(t *testing.T) {
	manifest := &api.RunManifest{
		Mounts: []api.Mount{{Name: "ns.team", GitRepoURL: "x"}},
		Agents: []api.AgentSpec{{ID: "ns.team", Mount: "ns.team", Cwd: "ns.team", Command: "claude", Args: []string{}, AgentType: "claude"}},
	}
	store := runplan.NewStore(t.TempDir())
	tmuxFake := &fakeTmux{}
	r := New(Deps{API: &fakeAPI{manifest: manifest}, Git: &fakeGit{}, Tmux: tmuxFake, Store: store})
	r.newID = func() (string, error) { return "rid", nil }

	if _, err := r.Start("ns", "team"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := r.Tell("rid", nil, "ns.team", "hello"); err != nil {
		t.Fatalf("Tell: %v", err)
	}
	plan, _ := store.Load("rid")
	if len(plan.Messages) != 1 || plan.Messages[0].Content != "hello" {
		t.Errorf("Messages: got %+v", plan.Messages)
	}
}

func TestTellUnknownAgent(t *testing.T) {
	manifest := &api.RunManifest{
		Mounts: []api.Mount{{Name: "ns.team", GitRepoURL: "x"}},
		Agents: []api.AgentSpec{{ID: "ns.team", Mount: "ns.team", Cwd: "ns.team", Command: "claude", Args: []string{}, AgentType: "claude"}},
	}
	store := runplan.NewStore(t.TempDir())
	r := New(Deps{API: &fakeAPI{manifest: manifest}, Git: &fakeGit{}, Tmux: &fakeTmux{}, Store: store})
	r.newID = func() (string, error) { return "rid", nil }
	if _, err := r.Start("ns", "team"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	err := r.Tell("rid", nil, "no.such.agent", "x")
	if !errors.Is(err, ErrAgentNotInRun) {
		t.Errorf("err: got %v, want ErrAgentNotInRun", err)
	}
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
