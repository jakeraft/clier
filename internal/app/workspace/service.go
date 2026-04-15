package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
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
	Kind          string `json:"kind"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	Local         string `json:"local"`
	PinnedVersion *int   `json:"pinned_version,omitempty"`
	LatestVersion *int   `json:"latest_version,omitempty"`
	Remote        string `json:"remote,omitempty"`
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

func NewService(client *api.Client, fs FileMaterializer, git GitRepo) *Service {
	return &Service{client: client, fs: fs, git: git}
}

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

func (s *Service) Pull(base string, force bool) (*Manifest, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	pulled, err := s.pullTarget(base, manifest, force)
	if err != nil {
		return nil, err
	}
	if err := SaveManifest(s.fs, base, pulled); err != nil {
		return nil, err
	}
	return pulled, nil
}

func (s *Service) pullTarget(base string, manifest *Manifest, force bool) (*Manifest, error) {
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

	// Pull each tracked resource's latest version from server.
	for i := range manifest.TrackedResources {
		tr := &manifest.TrackedResources[i]
		res, err := s.client.GetResource(tr.Owner, tr.Name)
		if err != nil {
			return nil, fmt.Errorf("pull %s %s/%s: %w", tr.Kind, tr.Owner, tr.Name, err)
		}
		localPath := filepath.Join(base, filepath.FromSlash(tr.LocalPath))
		latest := res.Metadata.LatestVersion

		switch api.ResourceKind(tr.Kind) {
		case api.KindMember:
			projection := memberProjectionFromResource(res)
			if err := WriteMemberProjection(s.fs, localPath, projection); err != nil {
				return nil, err
			}
		case api.KindTeam:
			teamSpec, err := api.DecodeSpec[api.TeamSpec](res)
			if err != nil {
				return nil, fmt.Errorf("decode team spec: %w", err)
			}
			if err := WriteTeamProjection(s.fs, localPath, teamProjectionFromResource(res, teamSpec)); err != nil {
				return nil, err
			}
		case api.KindClaudeMd, api.KindClaudeSettings, api.KindCodexMd, api.KindCodexSettings, api.KindSkill:
			spec, err := api.DecodeSpec[api.ContentSpec](res)
			if err != nil {
				return nil, fmt.Errorf("decode %s spec: %w", tr.Kind, err)
			}
			content := spec.Content
			if api.IsInstructionKind(tr.Kind) {
				memberName := filepath.ToSlash(tr.LocalPath)
				if idx := strings.Index(memberName, "/"); idx >= 0 {
					memberName = memberName[:idx]
				}
				agentType := s.resolveAgentTypeFromManifest(manifest, *tr)
				teamProtocol := s.loadTeamProtocolForMember(base, memberName)
				content = ComposeInstruction(agentType, memberName, content, teamProtocol)
			}
			if err := s.fs.EnsureFile(localPath, []byte(content)); err != nil {
				return nil, fmt.Errorf("write %s: %w", tr.LocalPath, err)
			}
		}

		tr.RemoteVersion = &latest
	}

	// Recalculate base hashes after updating files.
	if err := s.populateBaseHashes(base, manifest.TrackedResources); err != nil {
		return nil, err
	}

	return manifest, nil
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
	// Check remote versions — deduplicate by owner/name to avoid redundant API calls.
	type remoteVersion struct {
		latest int
		err    error
	}
	remoteCache := map[string]*remoteVersion{}
	for i, tr := range manifest.TrackedResources {
		if tr.RemoteVersion == nil {
			continue
		}
		key := tr.Owner + "/" + tr.Name
		if _, ok := remoteCache[key]; !ok {
			res, err := s.client.GetResource(tr.Owner, tr.Name)
			if err != nil {
				remoteCache[key] = &remoteVersion{err: err}
			} else {
				remoteCache[key] = &remoteVersion{latest: res.Metadata.LatestVersion}
			}
		}
		rv := remoteCache[key]
		if rv.err != nil {
			continue
		}
		pinned := *tr.RemoteVersion
		tracked[i].PinnedVersion = &pinned
		tracked[i].LatestVersion = &rv.latest
		if pinned < rv.latest {
			tracked[i].Remote = "behind"
		} else {
			tracked[i].Remote = "up-to-date"
		}
	}

	status := &Status{
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
	}

	return status, nil
}

func (s *Service) Push(base string) (*PushResult, error) {
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

	pushed := 0
	targetName := manifest.Name
	for _, resource := range modified {
		if !resource.Editable {
			continue
		}
		pushed++

		kind, body, err := s.preparePushBody(base, manifest, resource)
		if err != nil {
			return nil, err
		}
		updated, err := s.pushResource(resource, kind, body)
		if err != nil {
			return nil, err
		}
		if updated != nil && resource.LocalPath == manifest.RootResource.LocalPath {
			targetName = updated.Metadata.Name
		}
	}

	// Update manifest name if root resource was renamed, then pull latest.
	manifest.Name = targetName
	pulled, err := s.pullTarget(base, manifest, true)
	if err != nil {
		return nil, err
	}
	if err := SaveManifest(s.fs, base, pulled); err != nil {
		return nil, err
	}
	return &PushResult{Status: "pushed", Pushed: pushed, PulledAfterPush: true}, nil
}

// preparePushBody builds the request body for a single tracked resource.
func (s *Service) preparePushBody(base string, manifest *Manifest, r TrackedResource) (api.ResourceKind, any, error) {
	kind := api.ResourceKind(r.Kind)
	switch kind {
	case api.KindMember:
		projection, err := LoadMemberProjection(s.fs, filepath.Join(base, filepath.FromSlash(r.LocalPath)))
		if err != nil {
			return "", nil, err
		}
		agentType := s.resolveAgentTypeFromManifest(manifest, r)
		body, err := s.memberMutationFromProjection(projection, agentType)
		if err != nil {
			return "", nil, err
		}
		return kind, body, nil
	case api.KindTeam:
		projection, err := LoadTeamProjection(s.fs, filepath.Join(base, filepath.FromSlash(r.LocalPath)))
		if err != nil {
			return "", nil, err
		}
		body, err := s.teamMutationFromProjection(projection)
		if err != nil {
			return "", nil, err
		}
		return kind, body, nil
	default:
		content, err := s.readContentForPush(base, manifest, kind, r)
		if err != nil {
			return "", nil, err
		}
		return kind, api.ContentWriteRequest{Name: r.Name, Content: content}, nil
	}
}

// pushResource checks the remote version and uploads the resource.
func (s *Service) pushResource(r TrackedResource, kind api.ResourceKind, body any) (*api.ResourceResponse, error) {
	current, err := s.client.GetResource(r.Owner, r.Name)
	if err != nil {
		return nil, err
	}
	if !versionsMatch(r.RemoteVersion, current.Metadata.LatestVersion) {
		return nil, fmt.Errorf("remote %s %s/%s changed; pull before pushing", r.Kind, r.Owner, r.Name)
	}
	return s.client.UpdateResource(kind, r.Owner, r.Name, body)
}

// readContentForPush reads a content resource from disk, stripping the prelude for instruction kinds.
func (s *Service) readContentForPush(base string, manifest *Manifest, kind api.ResourceKind, r TrackedResource) (string, error) {
	if api.IsInstructionKind(string(kind)) {
		return s.serverInstructionContent(base, manifest, r)
	}
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(r.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", r.LocalPath, err)
	}
	return string(data), nil
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
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || name == ManifestFile || name == TeamProjectionFile {
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

func (s *Service) cloneMemberAsTeam(base, owner, name string) (*Manifest, error) {
	member, err := s.client.GetResource(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get member %s/%s: %w", owner, name, err)
	}
	projection := memberProjectionFromResource(member)
	agentType := agentTypeFromResource(member)

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

	// Write team projection (1-member team) — generated, not tracked.
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
	if err := WriteMemberProjection(s.fs, TeamMemberProjectionPath(base, member.Metadata.Name), projection); err != nil {
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

	profile, _ := domain.ProfileFor(agentType)

	generated := []string{
		TeamProjectionLocalPath(),
		filepath.ToSlash(filepath.Join(memberLocalBase, ".clier", "work-log-protocol.md")),
		filepath.ToSlash(filepath.Join(memberLocalBase, ".clier", TeamProtocolFileName(member.Metadata.Name))),
	}
	if profile.LocalSettingsFile != "" {
		generated = append(generated, filepath.ToSlash(filepath.Join(memberLocalBase, profile.SettingsDir, profile.LocalSettingsFile)))
	}

	if projection.InstructionRef != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          profile.InstructionKind,
			Owner:         projection.InstructionRef.Owner,
			Name:          projection.InstructionRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberLocalBase, profile.InstructionFile)),
			RemoteVersion: intPtr(projection.InstructionRef.Version),
			Editable:      true,
		})
	} else {
		generated = append(generated, filepath.ToSlash(filepath.Join(memberLocalBase, profile.InstructionFile)))
	}
	if projection.SettingsRef != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          profile.SettingsKind,
			Owner:         projection.SettingsRef.Owner,
			Name:          projection.SettingsRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberLocalBase, profile.SettingsDir, profile.SettingsFile)),
			RemoteVersion: intPtr(projection.SettingsRef.Version),
			Editable:      true,
		})
	}
	for _, skillRef := range projection.Skills {
		tracked = append(tracked, TrackedResource{
			Kind:          string(api.KindSkill),
			Owner:         skillRef.Owner,
			Name:          skillRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberLocalBase, profile.SettingsDir, profile.SkillsDir, skillRef.Name, "SKILL.md")),
			RemoteVersion: intPtr(skillRef.Version),
			Editable:      true,
		})
	}

	if err := s.populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}

	manifest := &Manifest{
		Kind:             string(api.KindMember),
		Owner:            member.Metadata.OwnerName,
		Name:             member.Metadata.Name,
		ClonedAt:         time.Now().UTC(),
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
		Kind:          string(api.KindTeam),
		Owner:         team.Metadata.OwnerName,
		Name:          team.Metadata.Name,
		LocalPath:     TeamProjectionLocalPath(),
		RemoteVersion: intPtr(team.Metadata.LatestVersion),
		Editable:      true,
	}}
	generated := []string{}
	metadata := &RuntimeMetadata{
		Team: &TeamRuntimeMetadata{
			ID:   team.Metadata.ID,
			Name: team.Metadata.Name,
		},
	}

	for _, tm := range refsByRelType(team, string(api.KindMember)) {
		memberVersion, err := s.client.GetResourceVersion(tm.OwnerName, tm.Name, tm.TargetVersion)
		if err != nil {
			return nil, fmt.Errorf("get member %s/%s: %w", tm.OwnerName, tm.Name, err)
		}

		// Build a MemberProjection from the snapshot + ref metadata.
		memberProjection := memberProjectionFromSnapshot(tm.Name, memberVersion)
		if err := WriteMemberProjection(s.fs, TeamMemberProjectionPath(base, tm.Name), memberProjection); err != nil {
			return nil, err
		}
		tracked = append(tracked, TrackedResource{
			Kind:          string(api.KindMember),
			Owner:         tm.OwnerName,
			Name:          tm.Name,
			LocalPath:     TeamMemberProjectionLocalPath(tm.Name),
			RemoteVersion: intPtr(tm.TargetVersion),
			Editable:      true,
		})

		agentType := agentTypeFromSnapshot(memberVersion.Snapshot, tm.AgentType)
		profile, _ := domain.ProfileFor(agentType)

		metadata.Team.Members = append(metadata.Team.Members, TeamMemberRuntimeMetadata{
			MemberID:   tm.TargetID,
			Name:       tm.Name,
			AgentType:  agentType,
			Command:    memberProjection.Command,
			GitRepoURL: memberProjection.GitRepoURL,
		})

		memberBase := filepath.ToSlash(tm.Name)
		generated = append(generated,
			filepath.ToSlash(filepath.Join(memberBase, ".clier", "work-log-protocol.md")),
			filepath.ToSlash(filepath.Join(memberBase, ".clier", TeamProtocolFileName(tm.Name))),
		)
		if profile.LocalSettingsFile != "" {
			generated = append(generated,
				filepath.ToSlash(filepath.Join(memberBase, profile.SettingsDir, profile.LocalSettingsFile)),
			)
		}

		if memberProjection.InstructionRef != nil {
			tracked = append(tracked, TrackedResource{
				Kind:          profile.InstructionKind,
				Owner:         memberProjection.InstructionRef.Owner,
				Name:          memberProjection.InstructionRef.Name,
				LocalPath:     filepath.ToSlash(filepath.Join(memberBase, profile.InstructionFile)),
				RemoteVersion: intPtr(memberProjection.InstructionRef.Version),
				Editable:      true,
			})
		} else {
			generated = append(generated, filepath.ToSlash(filepath.Join(memberBase, profile.InstructionFile)))
		}
		if memberProjection.SettingsRef != nil {
			tracked = append(tracked, TrackedResource{
				Kind:          profile.SettingsKind,
				Owner:         memberProjection.SettingsRef.Owner,
				Name:          memberProjection.SettingsRef.Name,
				LocalPath:     filepath.ToSlash(filepath.Join(memberBase, profile.SettingsDir, profile.SettingsFile)),
				RemoteVersion: intPtr(memberProjection.SettingsRef.Version),
				Editable:      true,
			})
		}
		for _, skillRef := range memberProjection.Skills {
			tracked = append(tracked, TrackedResource{
				Kind:          string(api.KindSkill),
				Owner:         skillRef.Owner,
				Name:          skillRef.Name,
				LocalPath:     filepath.ToSlash(filepath.Join(memberBase, profile.SettingsDir, profile.SkillsDir, skillRef.Name, "SKILL.md")),
				RemoteVersion: intPtr(skillRef.Version),
				Editable:      true,
			})
		}
	}

	if err := s.populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}
	manifest := &Manifest{
		Kind:             string(api.KindTeam),
		Owner:            team.Metadata.OwnerName,
		Name:             team.Metadata.Name,
		ClonedAt:         time.Now().UTC(),
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

func (s *Service) memberMutationFromProjection(projection *MemberProjection, agentType string) (*api.MemberWriteRequest, error) {
	profile, err := domain.ProfileFor(agentType)
	if err != nil {
		return nil, fmt.Errorf("resolve agent profile: %w", err)
	}

	var instructionRef *api.ResourceRefRequest
	if projection.InstructionRef != nil {
		res, err := s.client.GetResource(projection.InstructionRef.Owner, projection.InstructionRef.Name)
		if err != nil {
			return nil, err
		}
		instructionRef = &api.ResourceRefRequest{ID: res.Metadata.ID, Version: projection.InstructionRef.Version}
	}

	var settingsRef *api.ResourceRefRequest
	if projection.SettingsRef != nil {
		res, err := s.client.GetResource(projection.SettingsRef.Owner, projection.SettingsRef.Name)
		if err != nil {
			return nil, err
		}
		settingsRef = &api.ResourceRefRequest{ID: res.Metadata.ID, Version: projection.SettingsRef.Version}
	}

	skillRefs := make([]api.ResourceRefRequest, 0, len(projection.Skills))
	for _, skillRef := range projection.Skills {
		skill, err := s.client.GetResource(skillRef.Owner, skillRef.Name)
		if err != nil {
			return nil, err
		}
		skillRefs = append(skillRefs, api.ResourceRefRequest{ID: skill.Metadata.ID, Version: skillRef.Version})
	}

	req := &api.MemberWriteRequest{
		Name:       projection.Name,
		Command:    projection.Command,
		GitRepoURL: projection.GitRepoURL,
		Skills:     skillRefs,
	}
	req.SetInstructionRef(profile.InstructionKind, instructionRef)
	req.SetSettingsRef(profile.SettingsKind, settingsRef)
	return req, nil
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

// agentTypeFromResource resolves the agent type from a live ResourceResponse.
// The top-level AgentTypes field takes precedence over the spec-level agent_type.
func agentTypeFromResource(r *api.ResourceResponse) string {
	if len(r.AgentTypes) > 0 {
		return r.AgentTypes[0]
	}
	if spec, err := api.DecodeSpec[api.MemberSpec](r); err == nil {
		return spec.AgentType
	}
	return ""
}

// agentTypeFromSnapshot resolves the agent type from a version snapshot.
// The refAgentType (from the parent ref, e.g. team_member) takes precedence
// over the snapshot's embedded agent_type.
func agentTypeFromSnapshot(snapshot json.RawMessage, refAgentType string) string {
	if refAgentType != "" {
		return refAgentType
	}
	if spec, err := decodeSnapshot[api.MemberSpec](snapshot); err == nil {
		return spec.AgentType
	}
	return ""
}

// memberProjectionFromResource builds a MemberProjection from a unified ResourceResponse.
func memberProjectionFromResource(r *api.ResourceResponse) *MemberProjection {
	projection := &MemberProjection{
		Name:   r.Metadata.Name,
		Skills: make([]ResourceRefProjection, 0),
	}

	// Spec fields (Command, GitRepoURL) need to be decoded.
	if spec, err := api.DecodeSpec[api.MemberSpec](r); err == nil {
		projection.Command = spec.Command
		projection.GitRepoURL = spec.GitRepoURL
	}

	for _, kind := range []api.ResourceKind{api.KindClaudeMd, api.KindCodexMd} {
		if ref := firstRefByRelType(r, string(kind)); ref != nil {
			projection.InstructionRef = &ResourceRefProjection{Owner: ref.OwnerName, Name: ref.Name, Version: ref.TargetVersion}
			break
		}
	}
	for _, kind := range []api.ResourceKind{api.KindClaudeSettings, api.KindCodexSettings} {
		if ref := firstRefByRelType(r, string(kind)); ref != nil {
			projection.SettingsRef = &ResourceRefProjection{Owner: ref.OwnerName, Name: ref.Name, Version: ref.TargetVersion}
			break
		}
	}
	for _, ref := range refsByRelType(r, string(api.KindSkill)) {
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

	// Decode refs from snapshot. The server sends a "refs" array of objects
	// with rel_type, target_id, target_name, target_owner, target_version.
	type snapshotRef struct {
		RelType       string `json:"rel_type"`
		TargetID      int64  `json:"target_id"`
		TargetName    string `json:"target_name"`
		TargetOwner   string `json:"target_owner"`
		TargetVersion int    `json:"target_version"`
	}
	type snapshotRefs struct {
		Refs []snapshotRef `json:"refs"`
	}
	var refs snapshotRefs
	if err := decodeSnapshotInto(vr.Snapshot, &refs); err == nil {
		for _, ref := range refs.Refs {
			rp := ResourceRefProjection{
				Owner:   ref.TargetOwner,
				Name:    ref.TargetName,
				Version: ref.TargetVersion,
			}
			switch ref.RelType {
			case string(api.KindClaudeMd), string(api.KindCodexMd):
				projection.InstructionRef = &rp
			case string(api.KindClaudeSettings), string(api.KindCodexSettings):
				projection.SettingsRef = &rp
			case string(api.KindSkill):
				projection.Skills = append(projection.Skills, rp)
			}
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
	for _, ref := range refsByRelType(r, string(api.KindMember)) {
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

// resolveAgentTypeFromManifest finds the agent type for a tracked resource
// by checking which member's directory the resource's local path belongs to.
func (s *Service) resolveAgentTypeFromManifest(manifest *Manifest, r TrackedResource) string {
	if manifest.Runtime == nil || manifest.Runtime.Team == nil {
		return ""
	}
	localPath := filepath.ToSlash(filepath.Clean(r.LocalPath))
	for _, m := range manifest.Runtime.Team.Members {
		prefix := m.Name + "/"
		if strings.HasPrefix(localPath, prefix) {
			return m.AgentType
		}
	}
	return ""
}

func (s *Service) loadTeamProtocolForMember(base, memberName string) string {
	protocolPath := filepath.Join(base, memberName, ".clier", TeamProtocolFileName(memberName))
	data, err := s.fs.ReadFile(protocolPath)
	if err != nil {
		return ""
	}
	return string(data)
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

func versionsMatch(expected *int, actual int) bool {
	if expected == nil {
		return false
	}
	return *expected == actual
}

func (s *Service) serverInstructionContent(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
	}
	content := string(data)
	if manifest.Runtime != nil && manifest.Runtime.Team != nil {
		for _, member := range manifest.Runtime.Team.Members {
			profile, pErr := domain.ProfileFor(member.AgentType)
			if pErr != nil {
				continue
			}
			memberPath := filepath.ToSlash(filepath.Join(member.Name, profile.InstructionFile))
			clean := filepath.ToSlash(filepath.Clean(resource.LocalPath))
			if clean == memberPath {
				return StripInstructionPrelude(member.AgentType, member.Name, content), nil
			}
		}
	}
	return content, nil
}
