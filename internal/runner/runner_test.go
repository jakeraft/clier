package runner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/api"
	"github.com/jakeraft/clier/internal/runplan"
)

type fakeAPI struct {
	manifest *api.RunManifest
}

func (f *fakeAPI) MintRun(namespace, name string) (*api.RunManifest, error) {
	return f.manifest, nil
}

type fakeGit struct {
	clones []clone
}

type clone struct{ repoURL, dest string }

func (g *fakeGit) Clone(repoURL, dest string) error {
	g.clones = append(g.clones, clone{repoURL, dest})
	// Pretend the clone created the directory so any post-clone fs work
	// the runner does (none today, but defensive) sees it.
	_ = os.MkdirAll(dest, 0o755)
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

func soloAgentManifest(runID, id string) *api.RunManifest {
	return &api.RunManifest{
		RunID: runID,
		Agents: []api.AgentSpec{
			{
				ID: id,
				Prepare: api.AgentPrepare{
					Git: api.GitPrepare{
						RepoURL: "https://github.com/" + id,
						Subpath: "",
						Dest:    id,
					},
					Protocol: &api.ProtocolPrepare{
						Content: "# Team Protocol\n\nrendered body for " + id,
						Dest:    "protocols/" + id + ".md",
					},
				},
				Run: api.AgentRun{
					AgentType: "claude",
					Command:   "claude --setting-sources project",
					Args:      []string{"--append-system-prompt-file", "../protocols/" + id + ".md"},
				},
			},
		},
	}
}

func TestStartHappyPath_singleAgent(t *testing.T) {
	manifest := soloAgentManifest("rid-1", "jakeraft.hello-clier")

	root := t.TempDir()
	store := runplan.NewStore(root)
	gitFake := &fakeGit{}
	tmuxFake := &fakeTmux{}
	r := New(Deps{API: &fakeAPI{manifest: manifest}, Git: gitFake, Tmux: tmuxFake, Store: store})

	plan, err := r.Start("jakeraft", "hello-clier")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if plan.RunID != "rid-1" {
		t.Errorf("RunID: got %q, want rid-1", plan.RunID)
	}
	if plan.SessionName != "clier-rid-1" {
		t.Errorf("SessionName: got %q", plan.SessionName)
	}

	wantCloneDest := filepath.Join(store.RunDir("rid-1"), "jakeraft.hello-clier")
	if len(gitFake.clones) != 1 || gitFake.clones[0].dest != wantCloneDest {
		t.Errorf("clones: got %+v, want one clone to %q", gitFake.clones, wantCloneDest)
	}

	// Protocol file dropped at <run>/protocols/<id>.md verbatim.
	protoPath := filepath.Join(store.RunDir("rid-1"), "protocols", "jakeraft.hello-clier.md")
	body, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("read protocol file: %v", err)
	}
	if !strings.Contains(string(body), "rendered body for jakeraft.hello-clier") {
		t.Errorf("protocol file body: %q", string(body))
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
			if !strings.Contains(op.value, "claude --setting-sources project") {
				t.Errorf("send-keys missing command verbatim: %q", op.value)
			}
			if !strings.Contains(op.value, "--append-system-prompt-file ../protocols/jakeraft.hello-clier.md") {
				t.Errorf("send-keys missing file-path args: %q", op.value)
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
		RunID: "rid-2",
		Agents: []api.AgentSpec{
			{
				ID: "ns.a",
				Prepare: api.AgentPrepare{
					Git:      api.GitPrepare{RepoURL: "https://example/a", Dest: "ns.a"},
					Protocol: &api.ProtocolPrepare{Content: "p-a", Dest: "protocols/ns.a.md"},
				},
				Run: api.AgentRun{AgentType: "claude", Command: "claude", Args: []string{}},
			},
			{
				ID: "ns.b",
				Prepare: api.AgentPrepare{
					Git:      api.GitPrepare{RepoURL: "https://example/b", Dest: "ns.b"},
					Protocol: &api.ProtocolPrepare{Content: "p-b", Dest: "protocols/ns.b.md"},
				},
				Run: api.AgentRun{AgentType: "claude", Command: "claude", Args: []string{}},
			},
		},
	}
	r := New(Deps{
		API:   &fakeAPI{manifest: manifest},
		Git:   &fakeGit{},
		Tmux:  &fakeTmux{},
		Store: runplan.NewStore(t.TempDir()),
	})

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

// Codex agents arrive with prepare.protocol == nil because their
// rendered protocol travels inline in run.args. The runner must (a)
// skip the protocol file write entirely (no zero-byte file dropped on
// disk) and (b) auto-dismiss the vendor trust prompt by send-keys'ing
// the per-vendor trustResponse — for codex that's "1". Both are core
// to ADR-0002 §5/§7 and were the dominant new behavior in this PR, so
// the absence of either branch in earlier tests was a real coverage
// gap.
func TestStartHappyPath_codexNilProtocol(t *testing.T) {
	manifest := &api.RunManifest{
		RunID: "rid-cx",
		Agents: []api.AgentSpec{{
			ID: "jakeraft.hello-codex",
			Prepare: api.AgentPrepare{
				Git: api.GitPrepare{
					RepoURL: "https://github.com/jakeraft/hello-codex",
					Dest:    "jakeraft.hello-codex",
				},
				// Protocol intentionally nil — codex inlines the
				// protocol via run.args, so the manifest carries no
				// prepare.protocol block.
			},
			Run: api.AgentRun{
				AgentType: "codex",
				Command:   "codex --dangerously-bypass-approvals-and-sandbox",
				Args:      []string{"-c", "developer_instructions='''<rendered>'''"},
			},
		}},
	}

	store := runplan.NewStore(t.TempDir())
	tmuxFake := &fakeTmux{}
	r := New(Deps{
		API:   &fakeAPI{manifest: manifest},
		Git:   &fakeGit{},
		Tmux:  tmuxFake,
		Store: store,
	})

	plan, err := r.Start("jakeraft", "hello-codex")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// (a) No protocol file should land on disk for codex.
	protoPath := filepath.Join(store.RunDir("rid-cx"), "protocols", "jakeraft.hello-codex.md")
	if _, err := os.Stat(protoPath); !os.IsNotExist(err) {
		t.Errorf("expected no protocol file at %s, stat err: %v", protoPath, err)
	}
	// Plan record reflects the omission too.
	if got := plan.Agents[0].ProtocolDest; got != "" {
		t.Errorf("ProtocolDest: got %q, want empty", got)
	}

	// (b) Auto-trust send-keys ("1") should appear after the launch.
	// Op order: new-session, send (launch), send (trust "1").
	var sends []string
	for _, op := range tmuxFake.ops {
		if op.kind == "send" {
			sends = append(sends, op.value)
		}
	}
	if len(sends) != 2 {
		t.Fatalf("send-keys count: got %d, want 2 (launch + trust); ops=%+v", len(sends), tmuxFake.ops)
	}
	if sends[1] != "1" {
		t.Errorf("trust response: got %q, want %q", sends[1], "1")
	}
}

func TestTellRecordsMessage(t *testing.T) {
	manifest := soloAgentManifest("rid", "ns.team")
	store := runplan.NewStore(t.TempDir())
	tmuxFake := &fakeTmux{}
	r := New(Deps{API: &fakeAPI{manifest: manifest}, Git: &fakeGit{}, Tmux: tmuxFake, Store: store})

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
	manifest := soloAgentManifest("rid", "ns.team")
	store := runplan.NewStore(t.TempDir())
	r := New(Deps{API: &fakeAPI{manifest: manifest}, Git: &fakeGit{}, Tmux: &fakeTmux{}, Store: store})
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
