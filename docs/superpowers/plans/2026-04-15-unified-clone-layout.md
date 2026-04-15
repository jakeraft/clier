# Unified Clone Layout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify member and team clone layouts so member clones become 1-member teams internally, eliminating all member/team branching.

**Architecture:** Remove `MemberRuntimeMetadata` and all member-specific path/protocol functions. Member clones use `TeamRuntimeMetadata` with a single member. Clone/Pull/Push/Run converge on one code path. `manifest.Kind` stays as the server resource kind for API calls.

**Tech Stack:** Go, standard library only

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/app/workspace/manifest.go` | Modify | Remove `MemberRuntimeMetadata`, remove `RuntimeMetadata.Member` field |
| `internal/app/workspace/projection.go` | Modify | Remove `MemberProjectionPath`, `MemberProjectionLocalPath` |
| `internal/app/workspace/protocol.go` | Modify | Remove `ComposeMemberClaudeMd`, `StripMemberClaudeMdPrelude` |
| `internal/app/workspace/protocol_test.go` | Modify | Remove `TestComposeAndStripMemberClaudeMdPrelude`, add 1-member team protocol test |
| `internal/app/workspace/writer.go` | Modify | Remove `MaterializeMemberFiles`, always set `TeamMemberName` |
| `internal/app/workspace/service.go` | Modify | Replace `CloneMember`/`CloneTeam`/`materializeMember` with unified `Clone` |
| `internal/app/workspace/manifest_test.go` | Modify | Add test for member manifest with Team runtime |
| `cmd/clone.go` | Modify | Remove `cloneResolvedResource` switch, call `svc.Clone` |
| `cmd/run.go` | Modify | Remove `run start` switch, always use `Runtime.Team` |
| `cmd/working_copy_validation.go` | Modify | Remove member branch, always loop `Runtime.Team.Members` |
| `cmd/working_copy_validation_test.go` | Modify | Update test fixtures to use `Team` runtime |
| `cmd/working_copy_paths.go` | Modify | Remove `requireCurrentCopyRootKind` (no longer needed) |

---

### Task 1: Remove `MemberRuntimeMetadata` and unify `RuntimeMetadata`

**Files:**
- Modify: `internal/app/workspace/manifest.go:34-58`

- [ ] **Step 1: Delete `MemberRuntimeMetadata` type and `RuntimeMetadata.Member` field**

Change `manifest.go` from:

```go
type RuntimeMetadata struct {
	Member *MemberRuntimeMetadata `json:"member,omitempty"`
	Team   *TeamRuntimeMetadata   `json:"team,omitempty"`
}

type MemberRuntimeMetadata struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AgentType  string `json:"agent_type"`
	Command    string `json:"command"`
	GitRepoURL string `json:"git_repo_url,omitempty"`
}
```

To:

```go
type RuntimeMetadata struct {
	Team *TeamRuntimeMetadata `json:"team,omitempty"`
}
```

Delete the entire `MemberRuntimeMetadata` struct.

- [ ] **Step 2: Run build to identify all compilation errors**

Run: `go build ./...`
Expected: Compilation errors in files referencing `MemberRuntimeMetadata` and `RuntimeMetadata.Member`. Note every error location — these are the sites we fix in subsequent tasks.

- [ ] **Step 3: Commit**

```bash
git add internal/app/workspace/manifest.go
git commit -m "refactor: remove MemberRuntimeMetadata from manifest"
```

---

### Task 2: Remove member-specific projection path functions

**Files:**
- Modify: `internal/app/workspace/projection.go:42-56`

- [ ] **Step 1: Delete `MemberProjectionPath` and `MemberProjectionLocalPath`**

Remove these two functions from `projection.go`:

```go
func MemberProjectionPath(base string) string {
	return filepath.Join(base, ".clier", "member.json")
}

func MemberProjectionLocalPath() string {
	return filepath.ToSlash(filepath.Join(".clier", "member.json"))
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/app/workspace/projection.go
git commit -m "refactor: remove member-specific projection path functions"
```

---

### Task 3: Remove member-specific protocol functions

**Files:**
- Modify: `internal/app/workspace/protocol.go:43-62`
- Modify: `internal/app/workspace/protocol_test.go:84-97`

- [ ] **Step 1: Delete `ComposeMemberClaudeMd` and `StripMemberClaudeMdPrelude` from `protocol.go`**

Remove:

```go
func ComposeMemberClaudeMd(content string) string { ... }
func StripMemberClaudeMdPrelude(content string) string { ... }
```

- [ ] **Step 2: Delete `TestComposeAndStripMemberClaudeMdPrelude` from `protocol_test.go`**

Remove the entire test function.

- [ ] **Step 3: Run protocol tests**

Run: `go test ./internal/app/workspace/ -run TestCompose -v`
Expected: `TestComposeAndStripTeamClaudeMdPrelude` PASS. The deleted member test no longer runs.

- [ ] **Step 4: Commit**

```bash
git add internal/app/workspace/protocol.go internal/app/workspace/protocol_test.go
git commit -m "refactor: remove member-specific protocol functions"
```

---

### Task 4: Remove `MaterializeMemberFiles` and always set `TeamMemberName`

**Files:**
- Modify: `internal/app/workspace/writer.go:52-68,96-112`

- [ ] **Step 1: Delete `MaterializeMemberFiles` method**

Remove the entire `MaterializeMemberFiles` method (lines 52-68).

- [ ] **Step 2: Update `materializeMemberFiles` to always use team CLAUDE.md prelude**

In `materializeMemberFiles`, the `opts.TeamMemberName` check currently branches between `ComposeTeamClaudeMd` and `ComposeMemberClaudeMd`. Since `ComposeMemberClaudeMd` no longer exists and `TeamMemberName` is always set, simplify the CLAUDE.md write block.

Replace lines 96-112:

```go
	if projection.ClaudeMd != nil {
		// ... fetch content ...
		content := contentSpec.Content
		if opts.TeamMemberName != "" {
			content = ComposeTeamClaudeMd(opts.TeamMemberName, content)
		} else {
			content = ComposeMemberClaudeMd(content)
		}
		// ... write ...
	} else {
		content := ComposeMemberClaudeMd("")
		if opts.TeamMemberName != "" {
			content = ComposeTeamClaudeMd(opts.TeamMemberName, "")
		}
		// ... write ...
	}
```

With:

```go
	if projection.ClaudeMd != nil {
		vr, err := w.client.GetResourceVersion(projection.ClaudeMd.Owner, projection.ClaudeMd.Name, projection.ClaudeMd.Version)
		if err != nil {
			return fmt.Errorf("get claude md %s/%s: %w", projection.ClaudeMd.Owner, projection.ClaudeMd.Name, err)
		}
		contentSpec, err := decodeSnapshot[api.ContentSpec](vr.Snapshot)
		if err != nil {
			return fmt.Errorf("decode claude md %s/%s@%d: %w", projection.ClaudeMd.Owner, projection.ClaudeMd.Name, projection.ClaudeMd.Version, err)
		}
		content := ComposeTeamClaudeMd(opts.TeamMemberName, contentSpec.Content)
		if err := w.writeFile(paths.instructionFile, content); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	} else {
		content := ComposeTeamClaudeMd(opts.TeamMemberName, "")
		if err := w.writeFile(paths.instructionFile, content); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	}
```

- [ ] **Step 3: Commit**

```bash
git add internal/app/workspace/writer.go
git commit -m "refactor: remove MaterializeMemberFiles, always use team CLAUDE.md prelude"
```

---

### Task 5: Unify `Clone` in service layer

This is the largest task. Replace `CloneMember`, `CloneTeam`, and `materializeMember` with a single `Clone` method that wraps member resources into 1-member team structures.

**Files:**
- Modify: `internal/app/workspace/service.go:89-95,435-526,528-648,882-901`

- [ ] **Step 1: Write test for unified Clone with member kind**

Add to `internal/app/workspace/manifest_test.go`:

```go
func TestManifest_MemberCloneUsesTeamRuntime(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	resourceVersion := 1
	meta := &Manifest{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
		RootResource: TrackedResource{
			Kind:      string(api.KindMember),
			Owner:     "jakeraft",
			Name:      "reviewer",
			LocalPath: TeamMemberProjectionLocalPath("reviewer"),
			Editable:  true,
		},
		Runtime: &RuntimeMetadata{
			Team: &TeamRuntimeMetadata{
				ID:   0,
				Name: "reviewer",
				Members: []TeamMemberRuntimeMetadata{{
					MemberID:  42,
					Name:      "reviewer",
					AgentType: "claude",
					Command:   "claude",
				}},
			},
		},
		TrackedResources: []TrackedResource{{
			Kind:          string(api.KindMember),
			Owner:         "jakeraft",
			Name:          "reviewer",
			LocalPath:     TeamMemberProjectionLocalPath("reviewer"),
			RemoteVersion: &resourceVersion,
			Editable:      true,
		}},
	}

	if err := SaveManifest(filesystem.New(), base, meta); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	loaded, err := LoadManifest(filesystem.New(), base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.Runtime == nil || loaded.Runtime.Team == nil {
		t.Fatal("expected Team runtime metadata")
	}
	if len(loaded.Runtime.Team.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(loaded.Runtime.Team.Members))
	}
	if loaded.Runtime.Team.Members[0].Name != "reviewer" {
		t.Fatalf("member name = %q, want %q", loaded.Runtime.Team.Members[0].Name, "reviewer")
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/app/workspace/ -run TestManifest_MemberCloneUsesTeamRuntime -v`
Expected: PASS (this test only exercises manifest save/load with the new structure).

- [ ] **Step 3: Replace `CloneMember`, `CloneTeam` with unified `Clone`**

Remove `CloneMember` (line 89-91) and `CloneTeam` (line 93-95). Add:

```go
func (s *Service) Clone(base, kind, owner, name string) (*Manifest, error) {
	switch api.ResourceKind(kind) {
	case api.KindMember:
		return s.cloneMemberAsTeam(base, owner, name)
	case api.KindTeam:
		return s.materializeTeam(base, owner, name)
	default:
		return nil, fmt.Errorf("unsupported clone kind %q", kind)
	}
}
```

- [ ] **Step 4: Write `cloneMemberAsTeam` — wraps member into 1-member team and delegates to `materializeTeam` logic**

Add this new method that replaces `materializeMember`. It fetches the member resource, then builds all the structures that `materializeTeam` would build, but from a single member.

```go
func (s *Service) cloneMemberAsTeam(base, owner, name string) (*Manifest, error) {
	member, err := s.client.GetResource(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get member %s/%s: %w", owner, name, err)
	}
	projection := memberProjectionFromResource(member)
	agentType := agentTypeFromResource(member)

	// Write files using team layout: {base}/{memberName}/...
	writer := NewWriter(s.client, owner, s.fs, s.git)
	memberBase := filepath.Join(base, member.Metadata.Name)

	pinnedProjection, pinnedAgentType, err := writer.loadPinnedMember(
		owner, name, member.Metadata.LatestVersion, agentType,
	)
	if err != nil {
		return nil, fmt.Errorf("load pinned member %s: %w", name, err)
	}
	if err := writer.materializeMemberFiles(memberBase, pinnedProjection, pinnedAgentType, memberWriteOptions{
		TeamMemberName: member.Metadata.Name,
	}); err != nil {
		return nil, fmt.Errorf("materialize member %s: %w", name, err)
	}

	// Write team protocol for 1-member team (no relations).
	protocol := BuildAgentFacingTeamProtocol(
		member.Metadata.Name, member.Metadata.Name,
		domain.MemberRelations{Leaders: []int64{}, Workers: []int64{}},
		map[int64]ProtocolMember{member.Metadata.ID: {ID: member.Metadata.ID, Name: member.Metadata.Name}},
	)
	protocolPath := filepath.Join(memberBase, ".clier", TeamProtocolFileName(member.Metadata.Name))
	if err := s.fs.EnsureFile(protocolPath, []byte(protocol)); err != nil {
		return nil, fmt.Errorf("write protocol for %s: %w", name, err)
	}

	// Write team projection (1-member team).
	teamProjection := &TeamProjection{
		Name: member.Metadata.Name,
		Members: []TeamMemberProjection{{
			MemberID:      member.Metadata.ID,
			MemberVersion: member.Metadata.LatestVersion,
			Name:          member.Metadata.Name,
			Member: ResourceRefProjection{
				Owner:   member.Metadata.OwnerName,
				Name:    member.Metadata.Name,
				Version: member.Metadata.LatestVersion,
			},
		}},
		Relations: []TeamRelationProjection{},
	}
	if err := WriteTeamProjection(s.fs, TeamProjectionPath(base), teamProjection); err != nil {
		return nil, err
	}

	// Write member projection.
	if err := WriteMemberProjection(s.fs, TeamMemberProjectionPath(base, name), projection); err != nil {
		return nil, err
	}

	// Build tracked resources.
	memberLocalBase := filepath.ToSlash(member.Metadata.Name)
	tracked := []TrackedResource{{
		Kind:          string(api.KindMember),
		Owner:         member.Metadata.OwnerName,
		Name:          member.Metadata.Name,
		LocalPath:     TeamMemberProjectionLocalPath(member.Metadata.Name),
		RemoteVersion: intPtr(member.Metadata.LatestVersion),
		Editable:      true,
	}}

	generated := []string{
		TeamProjectionLocalPath(),
		filepath.ToSlash(filepath.Join(memberLocalBase, ".clier", "work-log-protocol.md")),
		filepath.ToSlash(filepath.Join(memberLocalBase, ".clier", TeamProtocolFileName(member.Metadata.Name))),
		filepath.ToSlash(filepath.Join(memberLocalBase, ".claude", "settings.local.json")),
	}

	if projection.ClaudeMd != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          string(api.KindClaudeMd),
			Owner:         projection.ClaudeMd.Owner,
			Name:          projection.ClaudeMd.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberLocalBase, "CLAUDE.md")),
			RemoteVersion: intPtr(projection.ClaudeMd.Version),
			Editable:      true,
		})
	} else {
		generated = append(generated, filepath.ToSlash(filepath.Join(memberLocalBase, "CLAUDE.md")))
	}
	if projection.ClaudeSettings != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          string(api.KindClaudeSettings),
			Owner:         projection.ClaudeSettings.Owner,
			Name:          projection.ClaudeSettings.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberLocalBase, ".claude", "settings.json")),
			RemoteVersion: intPtr(projection.ClaudeSettings.Version),
			Editable:      true,
		})
	}
	for _, skillRef := range projection.Skills {
		tracked = append(tracked, TrackedResource{
			Kind:          string(api.KindSkill),
			Owner:         skillRef.Owner,
			Name:          skillRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberLocalBase, ".claude", "skills", skillRef.Name, "SKILL.md")),
			RemoteVersion: intPtr(skillRef.Version),
			Editable:      true,
		})
	}

	if err := s.populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}

	manifest := &Manifest{
		Kind:     string(api.KindMember),
		Owner:    member.Metadata.OwnerName,
		Name:     member.Metadata.Name,
		ClonedAt: time.Now().UTC(),
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
		Runtime: &RuntimeMetadata{
			Team: &TeamRuntimeMetadata{
				ID:   0,
				Name: member.Metadata.Name,
				Members: []TeamMemberRuntimeMetadata{{
					MemberID:   member.Metadata.ID,
					Name:       member.Metadata.Name,
					AgentType:  agentType,
					Command:    projection.Command,
					GitRepoURL: projection.GitRepoURL,
				}},
			},
		},
	}
	if err := SaveManifest(s.fs, base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}
```

- [ ] **Step 5: Delete `materializeMember` method**

Remove the entire `materializeMember` function (lines 435-526).

- [ ] **Step 6: Add `domain` import if not already present**

Add `"github.com/jakeraft/clier/internal/domain"` to service.go imports.

- [ ] **Step 7: Update `pullTarget` — remove member projection branch, use team prelude for all CLAUDE.md**

In `pullTarget`, change the `KindMember` case (lines 139-142) to write to `TeamMemberProjectionPath`:

```go
case api.KindMember:
	projection := memberProjectionFromResource(res)
	// Extract member name from localPath: ".clier/members/{name}.json"
	memberName := filepath.Base(strings.TrimSuffix(tr.LocalPath, ".json"))
	if err := WriteMemberProjection(s.fs, localPath, projection); err != nil {
		return nil, err
	}
```

For the CLAUDE.md prelude (lines 159-170), remove the `manifest.Kind == string(api.KindTeam)` branch — always use `ComposeTeamClaudeMd`:

```go
if api.ResourceKind(tr.Kind) == api.KindClaudeMd {
	memberName := filepath.ToSlash(tr.LocalPath)
	if idx := strings.Index(memberName, "/"); idx >= 0 {
		memberName = memberName[:idx]
	}
	content = ComposeTeamClaudeMd(memberName, content)
}
```

- [ ] **Step 8: Update `preparePushBody` — change member projection path**

In `preparePushBody`, the `KindMember` case (lines 307-316) currently reads from `MemberProjectionPath`. Since member projections now live at `TeamMemberProjectionPath`, and the `TrackedResource.LocalPath` already points to the correct path, no change is needed — it already uses `filepath.Join(base, filepath.FromSlash(r.LocalPath))`.

Verify this is correct by reading the code. No actual edit needed.

- [ ] **Step 9: Update `serverClaudeMdContent` — always use `StripTeamClaudeMdPrelude`**

Replace the entire function body (lines 882-901):

```go
func (s *Service) serverClaudeMdContent(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
	}
	content := string(data)
	if manifest.Runtime != nil && manifest.Runtime.Team != nil {
		for _, member := range manifest.Runtime.Team.Members {
			memberPath := filepath.ToSlash(filepath.Join(member.Name, "CLAUDE.md"))
			clean := filepath.ToSlash(filepath.Clean(resource.LocalPath))
			if clean == memberPath {
				return StripTeamClaudeMdPrelude(member.Name, content), nil
			}
		}
	}
	return content, nil
}
```

- [ ] **Step 10: Run build**

Run: `go build ./...`
Expected: May still fail due to cmd/ package references. Those are fixed in subsequent tasks.

- [ ] **Step 11: Commit**

```bash
git add internal/app/workspace/service.go internal/app/workspace/manifest_test.go
git commit -m "refactor: unify Clone — member clones use team layout"
```

---

### Task 6: Update `cmd/clone.go`

**Files:**
- Modify: `cmd/clone.go:58-67`

- [ ] **Step 1: Replace `cloneResolvedResource` with direct `svc.Clone` call**

Remove the `cloneResolvedResource` function entirely. In `newCloneCmd`, replace:

```go
manifest, err := cloneResolvedResource(svc, base, kind, owner, name)
```

With:

```go
manifest, err := svc.Clone(base, kind, owner, name)
```

- [ ] **Step 2: Commit**

```bash
git add cmd/clone.go
git commit -m "refactor: use unified Clone in clone command"
```

---

### Task 7: Update `cmd/run.go` — remove run start switch

**Files:**
- Modify: `cmd/run.go:104-159`

- [ ] **Step 1: Replace the switch block in `newRunStartCmd` with unified team path**

Replace the entire `switch manifest.Kind` block (lines 104-159) with:

```go
			if manifest.Runtime == nil || manifest.Runtime.Team == nil {
				return fmt.Errorf("manifest is incomplete; pull the local clone again")
			}
			team := manifest.Runtime.Team
			runName := sessionName(team.Name, runID)
			var terminalPlans []apprun.MemberTerminal
			for i, member := range team.Members {
				memberProjection, err := appworkspace.LoadMemberProjection(fs, appworkspace.TeamMemberProjectionPath(copyRoot, member.Name))
				if err != nil {
					return err
				}
				memberBase := filepath.Join(copyRoot, member.Name)
				envVars := buildMemberEnv(runID, member.MemberID, &team.ID, member.Name)
				fullCommand := buildFullCommand(envVars, memberProjection.Command, memberBase)
				terminalPlans = append(terminalPlans, apprun.MemberTerminal{
					MemberID:    member.MemberID,
					Name:        member.Name,
					AgentType:   member.AgentType,
					Window:      i,
					Memberspace: memberBase,
					Cwd:         memberBase,
					Command:     fullCommand,
				})
			}
			runner := apprun.NewRunner(newTerminal())
			plan, err := runner.Run(copyRoot, runID, runName, terminalPlans)
			if err != nil {
				return err
			}
			return printJSON(map[string]any{"run_id": runID, "session": plan.Session})
```

- [ ] **Step 2: Remove unused `api` import if no longer needed**

Check if `api.KindMember` / `api.KindTeam` are still referenced in `run.go`. If not, remove the `api` import.

- [ ] **Step 3: Commit**

```bash
git add cmd/run.go
git commit -m "refactor: unify run start — remove member/team switch"
```

---

### Task 8: Update `cmd/working_copy_validation.go` and tests

**Files:**
- Modify: `cmd/working_copy_validation.go:17-49`
- Modify: `cmd/working_copy_validation_test.go`

- [ ] **Step 1: Rewrite `validateWorkingCopy` to always use Team runtime**

Replace the entire function:

```go
func validateWorkingCopy(base string, manifest *appworkspace.Manifest) error {
	if manifest == nil {
		return errors.New("working-copy manifest is missing")
	}
	if manifest.Runtime == nil || manifest.Runtime.Team == nil {
		return fmt.Errorf("manifest in %s is incomplete for runs", manifestPathLabel())
	}
	if len(manifest.Runtime.Team.Members) == 0 {
		return fmt.Errorf("manifest in %s is incomplete; pull the local clone again", manifestPathLabel())
	}
	for _, member := range manifest.Runtime.Team.Members {
		memberBase := filepath.Join(base, member.Name)
		if err := validateMemberCopy(memberBase, &appworkspace.TeamMemberRuntimeMetadata{
			MemberID:   member.MemberID,
			Name:       member.Name,
			Command:    member.Command,
			GitRepoURL: member.GitRepoURL,
		}, member.Name); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: Update `validateMemberCopy` signature to accept `TeamMemberRuntimeMetadata`**

```go
func validateMemberCopy(base string, member *appworkspace.TeamMemberRuntimeMetadata, teamMemberName string) error {
	if member == nil {
		return errors.New("working-copy member metadata is missing")
	}
	if member.MemberID == 0 || member.Name == "" || member.Command == "" {
		return fmt.Errorf("manifest in %s is incomplete; pull the local clone again", manifestPathLabel())
	}
	materialized, err := appworkspace.IsMaterializedRoot(newFileMaterializer(), newGitRepo(), member.GitRepoURL, base)
	if err != nil {
		return err
	}
	if !materialized {
		return fmt.Errorf("local clone is incomplete at %s", base)
	}
	required := []string{
		filepath.Join(base, "CLAUDE.md"),
		filepath.Join(base, ".clier", "work-log-protocol.md"),
		filepath.Join(base, ".claude", "settings.local.json"),
		filepath.Join(base, ".clier", appworkspace.TeamProtocolFileName(teamMemberName)),
	}
	for _, path := range required {
		if err := requireCopyPath(path); err != nil {
			return err
		}
	}
	return nil
}
```

Note: `teamMemberName` is always set now, so team protocol check always applies.

- [ ] **Step 3: Update tests to use Team runtime**

Replace `TestValidateWorkingCopy_Member`:

```go
func TestValidateWorkingCopy_Member(t *testing.T) {
	memberName := "reviewer"
	base := t.TempDir()
	memberBase := filepath.Join(base, memberName)
	required := []string{
		filepath.Join(memberBase, "CLAUDE.md"),
		filepath.Join(memberBase, ".clier", "work-log-protocol.md"),
		filepath.Join(memberBase, ".claude", "settings.local.json"),
		filepath.Join(memberBase, ".clier", appworkspace.TeamProtocolFileName(memberName)),
	}
	for _, path := range required {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	meta := &appworkspace.Manifest{
		Kind: string(api.KindMember),
		Runtime: &appworkspace.RuntimeMetadata{
			Team: &appworkspace.TeamRuntimeMetadata{
				ID:   0,
				Name: memberName,
				Members: []appworkspace.TeamMemberRuntimeMetadata{{
					MemberID: 1,
					Name:     memberName,
					Command:  "codex",
				}},
			},
		},
	}
	if err := validateWorkingCopy(base, meta); err != nil {
		t.Fatalf("validateWorkingCopy: %v", err)
	}
}
```

Replace `TestValidateWorkingCopy_MissingFileFails`:

```go
func TestValidateWorkingCopy_MissingFileFails(t *testing.T) {
	base := t.TempDir()
	meta := &appworkspace.Manifest{
		Kind: string(api.KindMember),
		Runtime: &appworkspace.RuntimeMetadata{
			Team: &appworkspace.TeamRuntimeMetadata{
				ID:   0,
				Name: "reviewer",
				Members: []appworkspace.TeamMemberRuntimeMetadata{{
					MemberID: 1,
					Name:     "reviewer",
					Command:  "codex",
				}},
			},
		},
	}
	if err := validateWorkingCopy(base, meta); err == nil {
		t.Fatalf("expected validation error for incomplete local clone")
	}
}
```

- [ ] **Step 4: Run validation tests**

Run: `go test ./cmd/ -run TestValidateWorkingCopy -v`
Expected: Both tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/working_copy_validation.go cmd/working_copy_validation_test.go
git commit -m "refactor: unify working copy validation — always use Team runtime"
```

---

### Task 9: Add 1-member team protocol test

**Files:**
- Modify: `internal/app/workspace/protocol_test.go`

- [ ] **Step 1: Add test for 1-member team protocol generation**

```go
func TestBuildAgentFacingTeamProtocol_SingleMemberTeam(t *testing.T) {
	protocol := BuildAgentFacingTeamProtocol(
		"reviewer",
		"reviewer",
		domain.MemberRelations{Leaders: []int64{}, Workers: []int64{}},
		map[int64]ProtocolMember{42: {ID: 42, Name: "reviewer"}},
	)

	if !strings.Contains(protocol, "You are **reviewer**, operating as a member of team **reviewer**.") {
		t.Fatalf("protocol should identify single member:\n%s", protocol)
	}
	if !strings.Contains(protocol, "- (none)") {
		t.Fatalf("protocol should show no relations:\n%s", protocol)
	}
}
```

- [ ] **Step 2: Run test**

Run: `go test ./internal/app/workspace/ -run TestBuildAgentFacingTeamProtocol_SingleMemberTeam -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/app/workspace/protocol_test.go
git commit -m "test: add 1-member team protocol generation test"
```

---

### Task 10: Clean up unused imports and final build verification

**Files:**
- Potentially modify: `cmd/run.go`, `cmd/clone.go`, `cmd/working_copy_validation.go`, `cmd/working_copy_paths.go`

- [ ] **Step 1: Run full build**

Run: `go build ./...`
Expected: PASS with no errors. If any unused import or dead code remains, fix it.

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`
Expected: All tests PASS.

- [ ] **Step 3: Fix any remaining issues**

Address any compilation errors or test failures from residual references to deleted types/functions.

- [ ] **Step 4: Commit if any cleanup was needed**

```bash
git add -A
git commit -m "refactor: clean up unused imports after clone layout unification"
```

---

### Task 11: Final verification

- [ ] **Step 1: Run full test suite one final time**

Run: `go test ./... -count=1`
Expected: All tests PASS.

- [ ] **Step 2: Run vet**

Run: `go vet ./...`
Expected: No issues.

- [ ] **Step 3: Review git log**

Run: `git log --oneline -10`
Expected: Clean commit history with incremental refactoring steps.
