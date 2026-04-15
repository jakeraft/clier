# Unified Clone Layout Design

## Problem

Member clone and team clone have different directory layouts, runtime metadata structures, and protocol generation. This causes member/team branching in 7+ files across clone, pull, push, run, and validation layers. The project's core value is declaring structure between members — standalone member run doesn't leverage this.

## Decision

Approach C: Keep `manifest.Kind` as the server resource kind (`"member"` / `"team"`), but unify the runtime structure so `RuntimeMetadata` always uses `TeamRuntimeMetadata`. A member clone becomes a 1-member team with no relations.

## Scope

- Server API: **no changes** — this is purely a CLI-side refactoring
- `manifest.Kind`: preserved as-is (matches server resource kind)
- `RuntimeMetadata.Member`: removed; member clone uses `RuntimeMetadata.Team` with 1 member

---

## 1. Manifest/Runtime Structure

### Before

```go
type RuntimeMetadata struct {
    Member *MemberRuntimeMetadata `json:"member,omitempty"`
    Team   *TeamRuntimeMetadata   `json:"team,omitempty"`
}
```

### After

```go
type RuntimeMetadata struct {
    Team *TeamRuntimeMetadata `json:"team,omitempty"`
}
```

`MemberRuntimeMetadata` type is deleted. For a member clone:
- `Team.ID` = 0 (no server-side team exists)
- `Team.Name` = member name
- `Team.Members` = single-element slice (the member itself)

---

## 2. Directory Layout

### Before (member clone)

```
{base}/CLAUDE.md
{base}/.clier/manifest.json
{base}/.clier/member.json
{base}/.clier/work-log-protocol.md
{base}/.claude/settings.local.json
```

### After (all clones, unified)

```
{base}/.clier/manifest.json
{base}/.clier/team.json
{base}/.clier/members/{memberName}.json
{base}/{memberName}/CLAUDE.md
{base}/{memberName}/.clier/work-log-protocol.md
{base}/{memberName}/.clier/{memberName}-team-protocol.md
{base}/{memberName}/.claude/settings.local.json
```

For a member clone, `team.json` contains a 1-member team projection and `members/` has one entry.

---

## 3. Clone/Materialize Unification

### Service layer

- Delete `CloneMember`, `CloneTeam` — replace with single `Clone(base, kind, owner, name)`
- Delete `materializeMember` — `materializeTeam` handles all clones
- `Clone` fetches from server based on Kind, then wraps member into 1-member team:
  - Kind == "member": fetch member resource, build `TeamProjection{Name: memberName, Members: [self], Relations: []}`, build `TeamRuntimeMetadata{ID: 0, Name: memberName, Members: [self]}`
  - Kind == "team": fetch team resource (current behavior)
- After wrapping, both paths call the same `materializeTeam`

### Writer layer

- Delete `MaterializeMemberFiles` — `MaterializeTeamFiles` handles all
- `materializeMemberFiles` (shared internal function) remains unchanged — it already works for both cases via `memberWriteOptions`
- `memberWriteOptions.TeamMemberName` is always set (no more empty-string case)

### cmd/clone.go

```go
// before
switch kind {
case "member": svc.CloneMember(...)
case "team":   svc.CloneTeam(...)
}

// after
svc.Clone(base, kind, owner, name)
```

---

## 4. Pull/Push

### Pull

`pullTarget` currently branches on `api.KindMember` vs `api.KindTeam` for projection writes:
- `KindMember` → `WriteMemberProjection` to `member.json`
- `KindTeam` → `WriteTeamProjection` to `team.json`

After unification:
- `KindMember` pull writes to `TeamMemberProjectionPath` (the single member entry under `members/`)
- `team.json` for member clones is a generated file (not tracked). It is derived from the member projection at clone time and does not need server-side pull. If the member name changes via push, the post-push pull (which re-clones via `pullTarget`) regenerates it.
- `KindTeam` pull writes `team.json` as before (tracked resource)
- CLAUDE.md prelude always uses `ComposeTeamClaudeMd`

### Push

`preparePushBody` branches on Kind for mutation construction:
- `KindMember` → `memberMutationFromProjection` — reads member projection, builds `MemberWriteRequest`
- `KindTeam` → `teamMutationFromProjection` — reads team projection, builds `TeamWriteRequest`

This branching stays because the server API expects different write payloads per Kind. But the projection source paths change:
- Member projection is now at `TeamMemberProjectionPath(base, name)` instead of `MemberProjectionPath(base)`

### serverClaudeMdContent (push CLAUDE.md stripping)

Currently branches between `StripMemberClaudeMdPrelude` and `StripTeamClaudeMdPrelude`. After unification, always uses `StripTeamClaudeMdPrelude` since all CLAUDE.md files have the team prelude format.

---

## 5. Run Start

### Before

```go
switch manifest.Kind {
case "member":
    // load MemberProjectionPath, manifest.Runtime.Member, teamID=nil, Cwd=copyRoot
case "team":
    // loop team.Members, TeamMemberProjectionPath, memberBase, &team.ID
}
```

### After

```go
team := manifest.Runtime.Team
runName := sessionName(team.Name, runID)
for i, member := range team.Members {
    memberProjection := loadMemberProjection(TeamMemberProjectionPath(copyRoot, member.Name))
    memberBase := filepath.Join(copyRoot, member.Name)
    envVars := buildMemberEnv(runID, member.MemberID, &team.ID, member.Name)
    // ... build MemberTerminal
}
```

The switch disappears. `team.ID` is 0 for member clones (no server-side team), but this is passed through to `CLIER_TEAM_ID` env var — agents can handle 0 as "no team".

### Other run subcommands

`run list`, `run view`, `run stop`, `run attach`, `run tell`, `run note` operate on `RunPlan` and have no member/team branching. No changes needed.

---

## 6. Validation

### Before

`validateWorkingCopy` branches on Kind: member validates `base` directly, team loops `member.Name` subdirectories.

### After

Always loops `manifest.Runtime.Team.Members` and validates `filepath.Join(base, member.Name)`. For member clones this loops once. `teamMemberName` is always set, so team protocol file check always applies.

---

## 7. Dead Code Removal

Types to delete:
- `MemberRuntimeMetadata`

Functions to delete:
- `CloneMember`, `CloneTeam` (replaced by `Clone`)
- `materializeMember` (absorbed into `materializeTeam`)
- `MaterializeMemberFiles` (absorbed into `MaterializeTeamFiles`)
- `ComposeMemberClaudeMd`, `StripMemberClaudeMdPrelude`
- `MemberProjectionPath`, `MemberProjectionLocalPath`

---

## 8. Projection Path Functions

### Before

```go
MemberProjectionPath(base)           → {base}/.clier/member.json
MemberProjectionLocalPath()          → .clier/member.json
TeamProjectionPath(base)             → {base}/.clier/team.json
TeamMemberProjectionPath(base, name) → {base}/.clier/members/{name}.json
TeamMemberProjectionLocalPath(name)  → .clier/members/{name}.json
```

### After

```go
TeamProjectionPath(base)             → {base}/.clier/team.json        (unchanged)
TeamMemberProjectionPath(base, name) → {base}/.clier/members/{name}.json (unchanged)
TeamMemberProjectionLocalPath(name)  → .clier/members/{name}.json     (unchanged)
```

`MemberProjectionPath` and `MemberProjectionLocalPath` are deleted.

---

## Files Changed

| File | Change |
|------|--------|
| `internal/app/workspace/manifest.go` | Remove `MemberRuntimeMetadata`, remove `RuntimeMetadata.Member` field |
| `internal/app/workspace/projection.go` | Remove `MemberProjectionPath`, `MemberProjectionLocalPath` |
| `internal/app/workspace/protocol.go` | Remove `ComposeMemberClaudeMd`, `StripMemberClaudeMdPrelude` |
| `internal/app/workspace/writer.go` | Remove `MaterializeMemberFiles`, always set `TeamMemberName` in options |
| `internal/app/workspace/service.go` | Remove `CloneMember`, `CloneTeam`, `materializeMember`; add `Clone`; update `pullTarget`, `preparePushBody`, `serverClaudeMdContent` |
| `cmd/clone.go` | Remove `cloneResolvedResource` switch; call `svc.Clone` |
| `cmd/run.go` | Remove `run start` switch; always use `Runtime.Team` |
| `cmd/working_copy_validation.go` | Remove member branch; always loop `Runtime.Team.Members` |
| `cmd/helpers.go` | Update `localPlanStore` if needed |
| `internal/app/workspace/*_test.go` | Update tests for unified structure |
| `cmd/*_test.go` | Update tests for unified structure |
