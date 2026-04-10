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

func currentUpstreamOfMember(member *api.MemberResponse) *UpstreamMetadata {
	if !member.IsFork || member.ForkOwnerLogin == "" || member.ForkName == "" {
		return nil
	}
	return &UpstreamMetadata{
		Kind:  "member",
		Owner: member.ForkOwnerLogin,
		Name:  member.ForkName,
	}
}

func currentUpstreamOfTeam(team *api.TeamResponse) *UpstreamMetadata {
	if !team.IsFork || team.ForkOwnerLogin == "" || team.ForkName == "" {
		return nil
	}
	return &UpstreamMetadata{
		Kind:  "team",
		Owner: team.ForkOwnerLogin,
		Name:  team.ForkName,
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

func NewService(client *api.Client) *Service {
	return &Service{client: client}
}

func (s *Service) CloneMember(base, owner, name string) (*Manifest, error) {
	return s.materializeMember(base, owner, name)
}

func (s *Service) CloneTeam(base, owner, name string) (*Manifest, error) {
	return s.materializeTeam(base, owner, name)
}

func (s *Service) Pull(base string, force bool) (*Manifest, error) {
	manifest, err := LoadManifest(base)
	if err != nil {
		return nil, err
	}
	pulled, err := s.pullTarget(base, manifest.Kind, manifest.Owner, manifest.Name, force)
	if err != nil {
		return nil, err
	}
	pulled.Upstream = preservedUpstreamState(manifest.Upstream, pulled.Upstream)
	if err := SaveManifest(base, pulled); err != nil {
		return nil, err
	}
	return pulled, nil
}

func (s *Service) pullTarget(base, kind, owner, name string, force bool) (*Manifest, error) {
	if !force {
		modified, err := ModifiedTrackedResources(base)
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
	manifest, err := LoadManifest(base)
	if err != nil {
		return nil, err
	}
	tracked, modifiedCount, err := trackedStatuses(base, manifest)
	if err != nil {
		return nil, err
	}
	runs, err := runSummary(base)
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
	manifest, err := LoadManifest(base)
	if err != nil {
		return nil, err
	}
	modified, err := ModifiedTrackedResources(base)
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
			projection, err := LoadMemberProjection(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, err
			}
			current, err := s.client.GetMember(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.LatestVersion) {
				return nil, fmt.Errorf("remote member %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			body, err := s.memberMutationFromProjection(projection)
			if err != nil {
				return nil, err
			}
			updated, err := s.client.UpdateMember(resource.Owner, resource.Name, body)
			if err != nil {
				return nil, err
			}
			if resource.LocalPath == manifest.RootResource.LocalPath {
				targetName = updated.Name
			}
		case "team":
			projection, err := LoadTeamProjection(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, err
			}
			current, err := s.client.GetTeam(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.LatestVersion) {
				return nil, fmt.Errorf("remote team %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			body, err := s.teamMutationFromProjection(projection)
			if err != nil {
				return nil, err
			}
			updated, err := s.client.UpdateTeam(resource.Owner, resource.Name, body)
			if err != nil {
				return nil, err
			}
			if resource.LocalPath == manifest.RootResource.LocalPath {
				targetName = updated.Name
			}
		case "claude-md":
			content, err := serverClaudeMdContent(base, manifest, resource)
			if err != nil {
				return nil, err
			}
			current, err := s.client.GetClaudeMd(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.LatestVersion) {
				return nil, fmt.Errorf("remote claude-md %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			if _, err := s.client.UpdateClaudeMd(resource.Owner, resource.Name, api.ClaudeMdWriteRequest{
				Name:    resource.Name,
				Content: content,
			}); err != nil {
				return nil, err
			}
		case "claude-settings":
			content, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
			}
			current, err := s.client.GetClaudeSettings(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.LatestVersion) {
				return nil, fmt.Errorf("remote claude-settings %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			if _, err := s.client.UpdateClaudeSettings(resource.Owner, resource.Name, api.ClaudeSettingsWriteRequest{
				Name:    resource.Name,
				Content: string(content),
			}); err != nil {
				return nil, err
			}
		case "skill":
			content, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
			if err != nil {
				return nil, fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
			}
			current, err := s.client.GetSkill(resource.Owner, resource.Name)
			if err != nil {
				return nil, err
			}
			if !versionsMatch(resource.RemoteVersion, current.LatestVersion) {
				return nil, fmt.Errorf("remote skill %s/%s changed; pull before pushing", resource.Owner, resource.Name)
			}
			if _, err := s.client.UpdateSkill(resource.Owner, resource.Name, api.SkillWriteRequest{
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

func ModifiedTrackedResources(base string) ([]TrackedResource, error) {
	manifest, err := LoadManifest(base)
	if err != nil {
		return nil, err
	}

	var modified []TrackedResource
	for _, resource := range manifest.TrackedResources {
		sum, err := fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
		if err != nil {
			return nil, err
		}
		if sum != resource.BaseHash {
			modified = append(modified, resource)
		}
	}
	return modified, nil
}

func trackedStatuses(base string, manifest *Manifest) ([]TrackedStatus, int, error) {
	statuses := make([]TrackedStatus, 0, len(manifest.TrackedResources))
	modifiedCount := 0
	for _, resource := range manifest.TrackedResources {
		sum, err := fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
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

func runSummary(base string) (RunStatusSummary, error) {
	dir := filepath.Join(base, ".clier")
	entries, err := os.ReadDir(dir)
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
	writer := NewWriter(s.client, owner)
	if err := writer.MaterializeMemberFiles(base, name); err != nil {
		return nil, err
	}

	member, err := s.client.GetMember(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get member %s/%s: %w", owner, name, err)
	}
	projection := memberProjectionFromResponse(member)
	if err := WriteMemberProjection(MemberProjectionPath(base), projection); err != nil {
		return nil, err
	}

	tracked := []TrackedResource{{
		Kind:          "member",
		Owner:         member.OwnerLogin,
		Name:          member.Name,
		LocalPath:     MemberProjectionLocalPath(),
		RemoteVersion: member.LatestVersion,
		Editable:      true,
	}}

	generated := []string{
		filepath.ToSlash(filepath.Join(".clier", "work-log-protocol.md")),
		filepath.ToSlash(filepath.Join(".claude", "settings.local.json")),
	}
	if member.ClaudeMd != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          "claude-md",
			Owner:         member.ClaudeMd.Owner,
			Name:          member.ClaudeMd.Name,
			LocalPath:     filepath.ToSlash("CLAUDE.md"),
			RemoteVersion: intPtr(member.ClaudeMd.Version),
			Editable:      true,
		})
	} else {
		generated = append(generated, filepath.ToSlash("CLAUDE.md"))
	}
	if member.ClaudeSettings != nil {
		tracked = append(tracked, TrackedResource{
			Kind:          "claude-settings",
			Owner:         member.ClaudeSettings.Owner,
			Name:          member.ClaudeSettings.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(".claude", "settings.json")),
			RemoteVersion: intPtr(member.ClaudeSettings.Version),
			Editable:      true,
		})
	}
	for _, skillRef := range member.Skills {
		tracked = append(tracked, TrackedResource{
			Kind:          "skill",
			Owner:         skillRef.Owner,
			Name:          skillRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(".claude", "skills", skillRef.Name, "SKILL.md")),
			RemoteVersion: intPtr(skillRef.Version),
			Editable:      true,
		})
	}
	if err := populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}

	manifest := &Manifest{
		Kind:             "member",
		Owner:            member.OwnerLogin,
		Name:             member.Name,
		ClonedAt:         time.Now().UTC(),
		Upstream:         currentUpstreamOfMember(member),
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
		Runtime: &RuntimeMetadata{
			Member: &MemberRuntimeMetadata{
				ID:         member.ID,
				Name:       member.Name,
				Command:    member.Command,
				GitRepoURL: member.GitRepoURL,
			},
		},
	}
	if err := SaveManifest(base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) materializeTeam(base, owner, name string) (*Manifest, error) {
	writer := NewWriter(s.client, owner)
	if err := writer.MaterializeTeamFiles(base, name); err != nil {
		return nil, err
	}

	team, err := s.client.GetTeam(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get team %s/%s: %w", owner, name, err)
	}
	if err := WriteTeamProjection(TeamProjectionPath(base), teamProjectionFromResponse(team)); err != nil {
		return nil, err
	}

	tracked := []TrackedResource{{
		Kind:          "team",
		Owner:         team.OwnerLogin,
		Name:          team.Name,
		LocalPath:     TeamProjectionLocalPath(),
		RemoteVersion: team.LatestVersion,
		Editable:      true,
	}}
	generated := []string{}
	metadata := &RuntimeMetadata{
		Team: &TeamRuntimeMetadata{
			ID:   team.ID,
			Name: team.Name,
		},
	}

	for _, tm := range team.TeamMembers {
		memberVersion, err := s.client.GetMemberVersion(tm.Member.Owner, tm.Member.Name, tm.Member.Version)
		if err != nil {
			return nil, fmt.Errorf("get member %s/%s: %w", tm.Member.Owner, tm.Member.Name, err)
		}
		memberSnapshot, err := loadMemberSnapshot(memberVersion.Content)
		if err != nil {
			return nil, fmt.Errorf("decode member %s/%s@%d: %w", tm.Member.Owner, tm.Member.Name, tm.Member.Version, err)
		}
		member := memberResponseFromSnapshot(tm.Member.Owner, tm.Member.Name, tm.Member.Version, memberSnapshot)
		if err := WriteMemberProjection(TeamMemberProjectionPath(base, tm.Name), memberProjectionFromResponse(member)); err != nil {
			return nil, err
		}
		tracked = append(tracked, TrackedResource{
			Kind:          "member",
			Owner:         tm.Member.Owner,
			Name:          tm.Member.Name,
			LocalPath:     TeamMemberProjectionLocalPath(tm.Name),
			RemoteVersion: intPtr(tm.Member.Version),
			Editable:      true,
		})
		metadata.Team.Members = append(metadata.Team.Members, TeamMemberRuntimeMetadata{
			TeamMemberID: tm.ID,
			Name:         tm.Name,
			Command:      memberSnapshot.Command,
			GitRepoURL:   memberSnapshot.GitRepoURL,
		})

		memberBase := filepath.ToSlash(tm.Name)
		generated = append(generated,
			filepath.ToSlash(filepath.Join(memberBase, ".clier", "work-log-protocol.md")),
			filepath.ToSlash(filepath.Join(memberBase, ".clier", TeamProtocolFileName(tm.Name))),
			filepath.ToSlash(filepath.Join(memberBase, ".claude", "settings.local.json")),
		)
		if memberSnapshot.ClaudeMd != nil {
			tracked = append(tracked, TrackedResource{
				Kind:          "claude-md",
				Owner:         memberSnapshot.ClaudeMd.Owner,
				Name:          memberSnapshot.ClaudeMd.Name,
				LocalPath:     filepath.ToSlash(filepath.Join(memberBase, "CLAUDE.md")),
				RemoteVersion: intPtr(memberSnapshot.ClaudeMd.Version),
				Editable:      true,
			})
		} else {
			generated = append(generated, filepath.ToSlash(filepath.Join(memberBase, "CLAUDE.md")))
		}
		if memberSnapshot.ClaudeSettings != nil {
			tracked = append(tracked, TrackedResource{
				Kind:          "claude-settings",
				Owner:         memberSnapshot.ClaudeSettings.Owner,
				Name:          memberSnapshot.ClaudeSettings.Name,
				LocalPath:     filepath.ToSlash(filepath.Join(memberBase, ".claude", "settings.json")),
				RemoteVersion: intPtr(memberSnapshot.ClaudeSettings.Version),
				Editable:      true,
			})
		}
		for _, skillRef := range memberSnapshot.Skills {
			tracked = append(tracked, TrackedResource{
				Kind:          "skill",
				Owner:         skillRef.Owner,
				Name:          skillRef.Name,
				LocalPath:     filepath.ToSlash(filepath.Join(memberBase, ".claude", "skills", skillRef.Name, "SKILL.md")),
				RemoteVersion: intPtr(skillRef.Version),
				Editable:      true,
			})
		}
	}

	if err := populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}
	manifest := &Manifest{
		Kind:             "team",
		Owner:            team.OwnerLogin,
		Name:             team.Name,
		ClonedAt:         time.Now().UTC(),
		Upstream:         currentUpstreamOfTeam(team),
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
		Runtime:          metadata,
	}
	if err := SaveManifest(base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) memberMutationFromProjection(projection *MemberProjection) (*api.MemberWriteRequest, error) {
	var claudeMdRef *api.ResourceRefRequest
	if projection.ClaudeMd != nil {
		claudeMd, err := s.client.GetClaudeMd(projection.ClaudeMd.Owner, projection.ClaudeMd.Name)
		if err != nil {
			return nil, err
		}
		claudeMdRef = &api.ResourceRefRequest{ID: claudeMd.ID, Version: projection.ClaudeMd.Version}
	}

	var claudeSettingsRef *api.ResourceRefRequest
	if projection.ClaudeSettings != nil {
		settings, err := s.client.GetClaudeSettings(projection.ClaudeSettings.Owner, projection.ClaudeSettings.Name)
		if err != nil {
			return nil, err
		}
		claudeSettingsRef = &api.ResourceRefRequest{ID: settings.ID, Version: projection.ClaudeSettings.Version}
	}

	skillRefs := make([]api.ResourceRefRequest, 0, len(projection.Skills))
	for _, skillRef := range projection.Skills {
		skill, err := s.client.GetSkill(skillRef.Owner, skillRef.Name)
		if err != nil {
			return nil, err
		}
		skillRefs = append(skillRefs, api.ResourceRefRequest{ID: skill.ID, Version: skillRef.Version})
	}

	return &api.MemberWriteRequest{
		Name:           projection.Name,
		AgentType:      projection.AgentType,
		Command:        projection.Command,
		GitRepoURL:     projection.GitRepoURL,
		ClaudeMd:       claudeMdRef,
		ClaudeSettings: claudeSettingsRef,
		Skills:         skillRefs,
	}, nil
}

func (s *Service) teamMutationFromProjection(projection *TeamProjection) (*api.TeamWriteRequest, error) {
	members := make([]api.TeamMemberRequest, 0, len(projection.Members))
	indicesByTeamMemberID := make(map[int64]int, len(projection.Members))
	rootIndex := -1
	for i, member := range projection.Members {
		resolved, err := s.client.GetMember(member.Member.Owner, member.Member.Name)
		if err != nil {
			return nil, err
		}
		members = append(members, api.TeamMemberRequest{
			Member: api.MemberRefRequest{
				ID:      resolved.ID,
				Version: member.Member.Version,
			},
			Name: member.Name,
		})
		indicesByTeamMemberID[member.TeamMemberID] = i
		if member.TeamMemberID == projection.RootTeamMemberID {
			rootIndex = i
		}
	}
	if rootIndex < 0 {
		return nil, fmt.Errorf("root team member id %d not found in team projection", projection.RootTeamMemberID)
	}

	relations := make([]api.TeamRelationRequest, 0, len(projection.Relations))
	for _, relation := range projection.Relations {
		fromIndex, ok := indicesByTeamMemberID[relation.FromTeamMemberID]
		if !ok {
			return nil, fmt.Errorf("relation source %d not found in team projection", relation.FromTeamMemberID)
		}
		toIndex, ok := indicesByTeamMemberID[relation.ToTeamMemberID]
		if !ok {
			return nil, fmt.Errorf("relation target %d not found in team projection", relation.ToTeamMemberID)
		}
		relations = append(relations, api.TeamRelationRequest{
			FromIndex: fromIndex,
			ToIndex:   toIndex,
		})
	}

	return &api.TeamWriteRequest{
		Name:        projection.Name,
		TeamMembers: members,
		Relations:   relations,
		RootIndex:   &rootIndex,
	}, nil
}

func memberProjectionFromResponse(member *api.MemberResponse) *MemberProjection {
	projection := &MemberProjection{
		Name:       member.Name,
		AgentType:  member.AgentType,
		Command:    member.Command,
		GitRepoURL: member.GitRepoURL,
		Skills:     make([]ResourceRefProjection, 0, len(member.Skills)),
	}
	if member.ClaudeMd != nil {
		projection.ClaudeMd = &ResourceRefProjection{Owner: member.ClaudeMd.Owner, Name: member.ClaudeMd.Name, Version: member.ClaudeMd.Version}
	}
	if member.ClaudeSettings != nil {
		projection.ClaudeSettings = &ResourceRefProjection{Owner: member.ClaudeSettings.Owner, Name: member.ClaudeSettings.Name, Version: member.ClaudeSettings.Version}
	}
	for _, skill := range member.Skills {
		projection.Skills = append(projection.Skills, ResourceRefProjection{Owner: skill.Owner, Name: skill.Name, Version: skill.Version})
	}
	return projection
}

func teamProjectionFromResponse(team *api.TeamResponse) *TeamProjection {
	projection := &TeamProjection{
		Name:      team.Name,
		Members:   make([]TeamMemberProjection, 0, len(team.TeamMembers)),
		Relations: make([]TeamRelationProjection, 0, len(team.Relations)),
	}
	if team.RootTeamMemberID != nil {
		projection.RootTeamMemberID = *team.RootTeamMemberID
	}
	for _, member := range team.TeamMembers {
		projection.Members = append(projection.Members, TeamMemberProjection{
			TeamMemberID: member.ID,
			Name:         member.Name,
			Member: ResourceRefProjection{
				Owner:   member.Member.Owner,
				Name:    member.Member.Name,
				Version: member.Member.Version,
			},
		})
	}
	for _, relation := range team.Relations {
		projection.Relations = append(projection.Relations, TeamRelationProjection{
			FromTeamMemberID: relation.FromTeamMemberID,
			ToTeamMemberID:   relation.ToTeamMemberID,
		})
	}
	return projection
}

func populateBaseHashes(base string, tracked []TrackedResource) error {
	for i := range tracked {
		sum, err := fileHash(filepath.Join(base, filepath.FromSlash(tracked[i].LocalPath)))
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

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
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

func serverClaudeMdContent(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	data, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
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
