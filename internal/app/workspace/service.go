package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	apprun "github.com/jakeraft/clier/internal/app/run"
)

type Service struct {
	client *api.Client
	fs     FileMaterializer
	git    GitRepo
}

type Status struct {
	WorkingCopy WorkingCopyStatus `json:"working_copy"`
	Local       string            `json:"local"`
	Tracked     []TrackedStatus   `json:"tracked"`
	Runs        RunStatusSummary  `json:"runs"`
}

type WorkingCopyStatus struct {
	Root     string    `json:"root"`
	Kind     string    `json:"kind"`
	Owner    string    `json:"owner"`
	Name     string    `json:"name"`
	ClonedAt time.Time `json:"cloned_at"`
}

type TrackedStatus struct {
	Kind  string `json:"kind"`
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	Local string `json:"local"`
}

type RunStatusSummary struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
}

type PushResult struct {
	Status          string `json:"status"`
	Pushed          int    `json:"pushed"`
	PulledAfterPush bool   `json:"pulled_after_push"`
}

// --- Resource helpers ---

// refsByRelType filters ResolvedRef entries from a ResourceResponse by rel_type.
func refsByRelType(r *api.ResourceResponse, relType string) []api.ResolvedRef {
	var out []api.ResolvedRef
	for _, ref := range r.Refs {
		if ref.RelType == relType {
			out = append(out, ref)
		}
	}
	return out
}

// firstRefByRelType returns the first ResolvedRef matching relType, or nil.
func firstRefByRelType(r *api.ResourceResponse, relType string) *api.ResolvedRef {
	for i := range r.Refs {
		if r.Refs[i].RelType == relType {
			return &r.Refs[i]
		}
	}
	return nil
}

func currentUpstreamOfResource(r *api.ResourceResponse, kind string) *UpstreamMetadata {
	if !r.Metadata.IsFork || r.Metadata.ForkOwnerName == "" || r.Metadata.ForkName == "" {
		return nil
	}
	return &UpstreamMetadata{
		Kind:  kind,
		Owner: r.Metadata.ForkOwnerName,
		Name:  r.Metadata.ForkName,
	}
}

func preservedUpstreamState(existing, current *UpstreamMetadata) *UpstreamMetadata {
	if current == nil {
		return nil
	}
	if existing == nil {
		return current
	}
	if existing.Kind != current.Kind || existing.Owner != current.Owner || existing.Name != current.Name {
		return current
	}
	current.FetchedVersion = existing.FetchedVersion
	current.FetchedAt = existing.FetchedAt
	return current
}

func NewService(client *api.Client, fs FileMaterializer, git GitRepo) *Service {
	return &Service{client: client, fs: fs, git: git}
}

func (s *Service) CloneMember(base, owner, name string) (*Manifest, error) {
	return s.materializeMember(base, owner, name)
}

func (s *Service) CloneTeam(base, owner, name string) (*Manifest, error) {
	return s.materializeTeam(base, owner, name)
}

func (s *Service) Pull(base string, force bool) (*Manifest, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	pulled, err := s.pullTarget(base, manifest.Kind, manifest.Owner, manifest.Name, force)
	if err != nil {
		return nil, err
	}
	pulled.Upstream = preservedUpstreamState(manifest.Upstream, pulled.Upstream)
	if err := SaveManifest(s.fs, base, pulled); err != nil {
		return nil, err
	}
	return pulled, nil
}

func (s *Service) pullTarget(base, kind, owner, name string, force bool) (*Manifest, error) {
	if !force {
		modified, err := s.ModifiedTrackedResources(base)
		if err != nil {
			return nil, err
		}
		if len(modified) > 0 {
			paths := make([]string, 0, len(modified))
			for _, resource := range modified {
				paths = append(paths, resource.LocalPath)
			}
			slices.Sort(paths)
			return nil, fmt.Errorf("local changes prevent pull; push or revert first: %s", strings.Join(paths, ", "))
		}
	}

	switch kind {
	case "member":
		return s.materializeMember(base, owner, name)
	case "team":
		return s.materializeTeam(base, owner, name)
	default:
		return nil, fmt.Errorf("unsupported working-copy kind %q", kind)
	}
}

func (s *Service) Status(base string) (*Status, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	tracked, modifiedCount, err := s.trackedStatuses(base, manifest)
	if err != nil {
		return nil, err
	}
	runs, err := s.runSummary(base)
	if err != nil {
		return nil, err
	}
	local := "clean"
	if modifiedCount > 0 {
		local = "modified"
	}
	return &Status{
		WorkingCopy: WorkingCopyStatus{
			Root:     base,
			Kind:     manifest.Kind,
			Owner:    manifest.Owner,
			Name:     manifest.Name,
			ClonedAt: manifest.ClonedAt,
		},
		Local:   local,
		Tracked: tracked,
		Runs:    runs,
	}, nil
}

func (s *Service) Push(base, currentLogin string) (*PushResult, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	modified, err := s.ModifiedTrackedResources(base)
	if err != nil {
		return nil, err
	}
	if len(modified) == 0 {
		return &PushResult{Status: "no_changes", Pushed: 0, PulledAfterPush: false}, nil
	}

	targetName := manifest.Name
	for _, resource := range modified {
		if !resource.Editable {
			continue
		}
		if resource.Owner != currentLogin {
			return nil, fmt.Errorf("cannot push %s %s/%s from %s: resource is not owned by %s",
				resource.Kind, resource.Owner, resource.Name, resource.LocalPath, currentLogin)
		}

		switch resource.Kind {
		case "member":
			projection, err := LoadMemberProjection(s.fs, filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, err
			}
			current, err := s.client.GetResource(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.Metadata.LatestVersion) {
				return nil, fmt.Errorf("remote member %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			body, err := s.memberMutationFromProjection(projection)
			if err != nil {
				return nil, err
			}
			updated, err := s.client.UpdateResource(api.KindMember, resource.Owner, resource.Name, body)
			if err != nil {
				return nil, err
			}
			if resource.LocalPath == manifest.RootResource.LocalPath {
				targetName = updated.Metadata.Name
			}
		case "team":
			projection, err := LoadTeamProjection(s.fs, filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, err
			}
			current, err := s.client.GetResource(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.Metadata.LatestVersion) {
				return nil, fmt.Errorf("remote team %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			body, err := s.teamMutationFromProjection(projection)
			if err != nil {
				return nil, err
			}
			updated, err := s.client.UpdateResource(api.KindTeam, resource.Owner, resource.Name, body)
			if err != nil {
				return nil, err
			}
			if resource.LocalPath == manifest.RootResource.LocalPath {
				targetName = updated.Metadata.Name
			}
		case "claude-md":
			content, err := s.serverClaudeMdContent(base, manifest, resource)
			if err != nil {
				return nil, err
			}
			current, err := s.client.GetResource(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.Metadata.LatestVersion) {
				return nil, fmt.Errorf("remote claude-md %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			if _, err := s.client.UpdateResource(api.KindClaudeMd, resource.Owner, resource.Name, api.ContentWriteRequest{
				Name:    resource.Name,
				Content: content,
			}); err != nil {
				return nil, err
			}
		case "claude-settings":
			content, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
			}
			current, err := s.client.GetResource(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.Metadata.LatestVersion) {
				return nil, fmt.Errorf("remote claude-settings %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			if _, err := s.client.UpdateResource(api.KindClaudeSettings, resource.Owner, resource.Name, api.ContentWriteRequest{
				Name:    resource.Name,
				Content: string(content),
			}); err != nil {
				return nil, err
			}
		case "skill":
			content, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
			}
			current, err := s.client.GetResource(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.Metadata.LatestVersion) {
				return nil, fmt.Errorf("remote skill %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			if _, err := s.client.UpdateResource(api.KindSkill, resource.Owner, resource.Name, api.ContentWriteRequest{
				Name:    resource.Name,
				Content: string(content),
			}); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported tracked resource kind %q", resource.Kind)
		}
	}

	if _, err := s.pullTarget(base, manifest.Kind, manifest.Owner, targetName, true); err != nil {
		return nil, err
	}
	return &PushResult{Status: "pushed", Pushed: len(modified), PulledAfterPush: true}, nil
}

func (s *Service) ModifiedTrackedResources(base string) ([]TrackedResource, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}

	var modified []TrackedResource
	for _, resource := range manifest.TrackedResources {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
		if err != nil {
			return nil, err
		}
		if sum != resource.BaseHash {
			modified = append(modified, resource)
		}
	}
	return modified, nil
}

func (s *Service) trackedStatuses(base string, manifest *Manifest) ([]TrackedStatus, int, error) {
	statuses := make([]TrackedStatus, 0, len(manifest.TrackedResources))
	modifiedCount := 0
	for _, resource := range manifest.TrackedResources {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
		if err != nil {
			return nil, 0, err
		}
		local := "clean"
		if sum != resource.BaseHash {
			local = "modified"
			modifiedCount++
		}
		statuses = append(statuses, TrackedStatus{
			Kind:  resource.Kind,
			Owner: resource.Owner,
			Name:  resource.Name,
			Path:  resource.LocalPath,
			Local: local,
		})
	}
	slices.SortFunc(statuses, func(a, b TrackedStatus) int {
		return strings.Compare(a.Path, b.Path)
	})
	return statuses, modifiedCount, nil
}

func (s *Service) runSummary(base string) (RunStatusSummary, error) {
	dir := filepath.Join(base, ".clier")
	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return RunStatusSummary{}, nil
		}
		return RunStatusSummary{}, fmt.Errorf("read runtime dir: %w", err)
	}
	var summary RunStatusSummary
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || name == ManifestFile {
			continue
		}
		plan, err := apprun.LoadPlanFromPath(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		summary.Total++
		if plan.Status == apprun.StatusRunning {
			summary.Running++
		} else {
			summary.Stopped++
		}
	}
	return summary, nil
}

func (s *Service) materializeMember(base, owner, name string) (*Manifest, error) {
	writer := NewWriter(s.client, owner, s.fs, s.git)
	if err := writer.MaterializeMemberFiles(base, name); err != nil {
		return nil, err
	}

	member, err := s.client.GetResource(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get member %s/%s: %w", owner, name, err)
	}
	memberSpec, err := api.DecodeSpec[api.MemberSpec](member)
	if err != nil {
		return nil, fmt.Errorf("decode member spec %s/%s: %w", owner, name, err)
	}
	projection := memberProjectionFromResource(member)
	if err := WriteMemberProjection(s.fs, MemberProjectionPath(base), projection); err != nil {
		return nil, err
	}

	tracked := []TrackedResource{{
		Kind:          "member",
		Owner:         member.Metadata.OwnerName,
		Name:          member.Metadata.Name,
		LocalPath:     MemberProjectionLocalPath(),
		RemoteVersion: member.Metadata.LatestVersion,
		Editable:      true,
	}}

	generated := []string{
		filepath.ToSlash(filepath.Join(".clier", "work-log-protocol.md")),
		filepath.ToSlash(filepath.Join(".claude", "settings.local.json")),
	}

	claudeMdRef := firstRefByRelType(member, "claude_md")
	if claudeMdRef != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          "claude-md",
			Owner:         claudeMdRef.OwnerName,
			Name:          claudeMdRef.Name,
			LocalPath:     filepath.ToSlash("CLAUDE.md"),
			RemoteVersion: intPtr(claudeMdRef.TargetVersion),
			Editable:      true,
		})
	} else {
		generated = append(generated, filepath.ToSlash("CLAUDE.md"))
	}

	claudeSettingsRef := firstRefByRelType(member, "claude_settings")
	if claudeSettingsRef != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          "claude-settings",
			Owner:         claudeSettingsRef.OwnerName,
			Name:          claudeSettingsRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(".claude", "settings.json")),
			RemoteVersion: intPtr(claudeSettingsRef.TargetVersion),
			Editable:      true,
		})
	}

	for _, skillRef := range refsByRelType(member, "skill") {
		tracked = append(tracked, TrackedResource{
			Kind:          "skill",
			Owner:         skillRef.OwnerName,
			Name:          skillRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(".claude", "skills", skillRef.Name, "SKILL.md")),
			RemoteVersion: intPtr(skillRef.TargetVersion),
			Editable:      true,
		})
	}
	if err := s.populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}

	// Determine agent type: use first from AgentTypes or fallback to spec.
	agentType := memberSpec.AgentType
	if len(member.AgentTypes) > 0 {
		agentType = member.AgentTypes[0]
	}

	manifest := &Manifest{
		Kind:     "member",
		Owner:    member.Metadata.OwnerName,
		Name:     member.Metadata.Name,
		ClonedAt: time.Now().UTC(),
		Upstream: currentUpstreamOfResource(member, "member"),
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
		Runtime: &RuntimeMetadata{
			Member: &MemberRuntimeMetadata{
				ID:         member.Metadata.ID,
				Name:       member.Metadata.Name,
				AgentType:  agentType,
				Command:    memberSpec.Command,
				GitRepoURL: memberSpec.GitRepoURL,
			},
		},
	}
	if err := SaveManifest(s.fs, base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) materializeTeam(base, owner, name string) (*Manifest, error) {
	writer := NewWriter(s.client, owner, s.fs, s.git)
	if err := writer.MaterializeTeamFiles(base, name); err != nil {
		return nil, err
	}

	team, err := s.client.GetResource(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get team %s/%s: %w", owner, name, err)
	}
	teamSpec, err := api.DecodeSpec[api.TeamSpec](team)
	if err != nil {
		return nil, fmt.Errorf("decode team spec %s/%s: %w", owner, name, err)
	}
	if err := WriteTeamProjection(s.fs, TeamProjectionPath(base), teamProjectionFromResource(team, teamSpec)); err != nil {
		return nil, err
	}

	tracked := []TrackedResource{{
		Kind:          "team",
		Owner:         team.Metadata.OwnerName,
		Name:          team.Metadata.Name,
		LocalPath:     TeamProjectionLocalPath(),
		RemoteVersion: team.Metadata.LatestVersion,
		Editable:      true,
	}}
	generated := []string{}
	metadata := &RuntimeMetadata{
		Team: &TeamRuntimeMetadata{
			ID:   team.Metadata.ID,
			Name: team.Metadata.Name,
		},
	}

	for _, tm := range refsByRelType(team, "team_member") {
		memberVersion, err := s.client.GetResourceVersion(tm.OwnerName, tm.Name, tm.TargetVersion)
		if err != nil {
			return nil, fmt.Errorf("get member %s/%s: %w", tm.OwnerName, tm.Name, err)
		}
		memberSnapshot, err := decodeSnapshot[api.MemberSpec](memberVersion.Snapshot)
		if err != nil {
			return nil, fmt.Errorf("decode member %s/%s@%d: %w", tm.OwnerName, tm.Name, tm.TargetVersion, err)
		}

		// Build a MemberProjection from the snapshot + ref metadata.
		memberProjection := memberProjectionFromSnapshot(tm.Name, memberVersion)
		if err := WriteMemberProjection(s.fs, TeamMemberProjectionPath(base, tm.Name), memberProjection); err != nil {
			return nil, err
		}
		tracked = append(tracked, TrackedResource{
			Kind:          "member",
			Owner:         tm.OwnerName,
			Name:          tm.Name,
			LocalPath:     TeamMemberProjectionLocalPath(tm.Name),
			RemoteVersion: intPtr(tm.TargetVersion),
			Editable:      true,
		})

		// Determine agent type from snapshot or ref.
		agentType := memberSnapshot.AgentType
		if tm.AgentType != "" {
			agentType = tm.AgentType
		}
		metadata.Team.Members = append(metadata.Team.Members, TeamMemberRuntimeMetadata{
			TeamMemberID: tm.ID,
			Name:         tm.Name,
			AgentType:    agentType,
			Command:      memberSnapshot.Command,
			GitRepoURL:   memberSnapshot.GitRepoURL,
		})

		memberBase := filepath.ToSlash(tm.Name)
		generated = append(generated,
			filepath.ToSlash(filepath.Join(memberBase, ".clier", "work-log-protocol.md")),
			filepath.ToSlash(filepath.Join(memberBase, ".clier", TeamProtocolFileName(tm.Name))),
			filepath.ToSlash(filepath.Join(memberBase, ".claude", "settings.local.json")),
		)

		// Extract content refs from the member version snapshot.
		// The snapshot has refs embedded; we decode them from the full snapshot.
		type memberSnapshotRefs struct {
			ClaudeMd       *ResourceRefProjection  `json:"claude_md,omitempty"`
			ClaudeSettings *ResourceRefProjection  `json:"claude_settings,omitempty"`
			Skills         []ResourceRefProjection `json:"skills,omitempty"`
		}
		var snapshotRefs memberSnapshotRefs
		if err := decodeSnapshotInto(memberVersion.Snapshot, &snapshotRefs); err == nil {
			if snapshotRefs.ClaudeMd != nil {
				tracked = append(tracked, TrackedResource{
					Kind:          "claude-md",
					Owner:         snapshotRefs.ClaudeMd.Owner,
					Name:          snapshotRefs.ClaudeMd.Name,
					LocalPath:     filepath.ToSlash(filepath.Join(memberBase, "CLAUDE.md")),
					RemoteVersion: intPtr(snapshotRefs.ClaudeMd.Version),
					Editable:      true,
				})
			} else {
				generated = append(generated, filepath.ToSlash(filepath.Join(memberBase, "CLAUDE.md")))
			}
			if snapshotRefs.ClaudeSettings != nil {
				tracked = append(tracked, TrackedResource{
					Kind:          "claude-settings",
					Owner:         snapshotRefs.ClaudeSettings.Owner,
					Name:          snapshotRefs.ClaudeSettings.Name,
					LocalPath:     filepath.ToSlash(filepath.Join(memberBase, ".claude", "settings.json")),
					RemoteVersion: intPtr(snapshotRefs.ClaudeSettings.Version),
					Editable:      true,
				})
			}
			for _, skillRef := range snapshotRefs.Skills {
				tracked = append(tracked, TrackedResource{
					Kind:          "skill",
					Owner:         skillRef.Owner,
					Name:          skillRef.Name,
					LocalPath:     filepath.ToSlash(filepath.Join(memberBase, ".claude", "skills", skillRef.Name, "SKILL.md")),
					RemoteVersion: intPtr(skillRef.Version),
					Editable:      true,
				})
			}
		} else {
			generated = append(generated, filepath.ToSlash(filepath.Join(memberBase, "CLAUDE.md")))
		}
	}

	if err := s.populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}
	manifest := &Manifest{
		Kind:             "team",
		Owner:            team.Metadata.OwnerName,
		Name:             team.Metadata.Name,
		ClonedAt:         time.Now().UTC(),
		Upstream:         currentUpstreamOfResource(team, "team"),
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
		Runtime:          metadata,
	}
	if err := SaveManifest(s.fs, base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) memberMutationFromProjection(projection *MemberProjection) (*api.MemberWriteRequest, error) {
	var claudeMdRef *api.ResourceRefRequest
	if projection.ClaudeMd != nil {
		claudeMd, err := s.client.GetResource(projection.ClaudeMd.Owner, projection.ClaudeMd.Name)
		if err != nil {
			return nil, err
		}
		claudeMdRef = &api.ResourceRefRequest{ID: claudeMd.Metadata.ID, Version: projection.ClaudeMd.Version}
	}

	var claudeSettingsRef *api.ResourceRefRequest
	if projection.ClaudeSettings != nil {
		settings, err := s.client.GetResource(projection.ClaudeSettings.Owner, projection.ClaudeSettings.Name)
		if err != nil {
			return nil, err
		}
		claudeSettingsRef = &api.ResourceRefRequest{ID: settings.Metadata.ID, Version: projection.ClaudeSettings.Version}
	}

	skillRefs := make([]api.ResourceRefRequest, 0, len(projection.Skills))
	for _, skillRef := range projection.Skills {
		skill, err := s.client.GetResource(skillRef.Owner, skillRef.Name)
		if err != nil {
			return nil, err
		}
		skillRefs = append(skillRefs, api.ResourceRefRequest{ID: skill.Metadata.ID, Version: skillRef.Version})
	}

	return &api.MemberWriteRequest{
		Name:           projection.Name,
		Command:        projection.Command,
		GitRepoURL:     projection.GitRepoURL,
		ClaudeMd:       claudeMdRef,
		ClaudeSettings: claudeSettingsRef,
		Skills:         skillRefs,
	}, nil
}

func (s *Service) teamMutationFromProjection(projection *TeamProjection) (*api.TeamWriteRequest, error) {
	members := make([]api.TeamMemberRequest, 0, len(projection.Members))
	for _, member := range projection.Members {
		resolved, err := s.client.GetResource(member.Member.Owner, member.Member.Name)
		if err != nil {
			return nil, err
		}
		members = append(members, api.TeamMemberRequest{
			MemberID:      resolved.Metadata.ID,
			MemberVersion: member.Member.Version,
		})
	}

	relations := make([]api.TeamRelationRequest, 0, len(projection.Relations))
	for _, relation := range projection.Relations {
		relations = append(relations, api.TeamRelationRequest{
			From: relation.From,
			To:   relation.To,
		})
	}

	return &api.TeamWriteRequest{
		Name:        projection.Name,
		TeamMembers: members,
		Relations:   relations,
	}, nil
}

// memberProjectionFromResource builds a MemberProjection from a unified ResourceResponse.
func memberProjectionFromResource(r *api.ResourceResponse) *MemberProjection {
	projection := &MemberProjection{
		Name:    r.Metadata.Name,
		Skills:  make([]ResourceRefProjection, 0),
	}

	// Spec fields (Command, GitRepoURL) need to be decoded.
	if spec, err := api.DecodeSpec[api.MemberSpec](r); err == nil {
		projection.Command = spec.Command
		projection.GitRepoURL = spec.GitRepoURL
	}

	if ref := firstRefByRelType(r, "claude_md"); ref != nil {
		projection.ClaudeMd = &ResourceRefProjection{Owner: ref.OwnerName, Name: ref.Name, Version: ref.TargetVersion}
	}
	if ref := firstRefByRelType(r, "claude_settings"); ref != nil {
		projection.ClaudeSettings = &ResourceRefProjection{Owner: ref.OwnerName, Name: ref.Name, Version: ref.TargetVersion}
	}
	for _, ref := range refsByRelType(r, "skill") {
		projection.Skills = append(projection.Skills, ResourceRefProjection{Owner: ref.OwnerName, Name: ref.Name, Version: ref.TargetVersion})
	}
	return projection
}

// memberProjectionFromSnapshot builds a MemberProjection from a version snapshot.
func memberProjectionFromSnapshot(name string, vr *api.ResourceVersionResponse) *MemberProjection {
	projection := &MemberProjection{
		Name:   name,
		Skills: make([]ResourceRefProjection, 0),
	}

	// Decode spec fields from snapshot.
	if spec, err := decodeSnapshot[api.MemberSpec](vr.Snapshot); err == nil {
		projection.Command = spec.Command
		projection.GitRepoURL = spec.GitRepoURL
	}

	// Decode refs from snapshot (they are embedded in the snapshot JSON).
	type snapshotRefs struct {
		ClaudeMd       *ResourceRefProjection  `json:"claude_md,omitempty"`
		ClaudeSettings *ResourceRefProjection  `json:"claude_settings,omitempty"`
		Skills         []ResourceRefProjection `json:"skills,omitempty"`
	}
	var refs snapshotRefs
	if err := decodeSnapshotInto(vr.Snapshot, &refs); err == nil {
		projection.ClaudeMd = refs.ClaudeMd
		projection.ClaudeSettings = refs.ClaudeSettings
		if len(refs.Skills) > 0 {
			projection.Skills = refs.Skills
		}
	}

	return projection
}

// teamProjectionFromResource builds a TeamProjection from a unified ResourceResponse.
func teamProjectionFromResource(r *api.ResourceResponse, spec *api.TeamSpec) *TeamProjection {
	projection := &TeamProjection{
		Name:      r.Metadata.Name,
		Members:   make([]TeamMemberProjection, 0),
		Relations: make([]TeamRelationProjection, 0),
	}
	for _, ref := range refsByRelType(r, "team_member") {
		projection.Members = append(projection.Members, TeamMemberProjection{
			MemberID:      ref.TargetID,
			MemberVersion: ref.TargetVersion,
			Name:          ref.Name,
			Member: ResourceRefProjection{
				Owner:   ref.OwnerName,
				Name:    ref.Name,
				Version: ref.TargetVersion,
			},
		})
	}
	for _, relation := range spec.Relations {
		projection.Relations = append(projection.Relations, TeamRelationProjection{
			From: relation.From,
			To:   relation.To,
		})
	}
	return projection
}

func (s *Service) populateBaseHashes(base string, tracked []TrackedResource) error {
	for i := range tracked {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(tracked[i].LocalPath)))
		if err != nil {
			return err
		}
		tracked[i].BaseHash = sum
	}
	return nil
}

func intPtr(v int) *int {
	return &v
}

func normalizePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, filepath.ToSlash(filepath.Clean(path)))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func (s *Service) fileHash(path string) (string, error) {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func versionsMatch(expected, actual *int) bool {
	switch {
	case expected == nil && actual == nil:
		return true
	case expected == nil || actual == nil:
		return false
	default:
		return *expected == *actual
	}
}

func (s *Service) serverClaudeMdContent(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
	}
	content := string(data)
	clean := filepath.ToSlash(filepath.Clean(resource.LocalPath))
	if clean == filepath.ToSlash("CLAUDE.md") {
		return StripMemberClaudeMdPrelude(content), nil
	}
	if manifest.Runtime != nil && manifest.Runtime.Team != nil {
		for _, member := range manifest.Runtime.Team.Members {
			memberPath := filepath.ToSlash(filepath.Join(member.Name, "CLAUDE.md"))
			if clean == memberPath {
				return StripTeamClaudeMdPrelude(member.Name, content), nil
			}
		}
	}
	return content, nil
}
