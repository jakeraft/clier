package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
	storemanifest "github.com/jakeraft/clier/internal/store/manifest"
)

type Service struct {
	client RemoteWorkspaceClient
	fs     FileMaterializer
	git    GitRepo
}

type Status struct {
	WorkingCopy WorkingCopyStatus
	Local       string
	Summary     StatusSummary
	Tracked     []TrackedStatus
	Runs        RunStatusSummary
}

type WorkingCopyStatus struct {
	Root     string
	Kind     string
	Owner    string
	Name     string
	ClonedAt time.Time
}

type TrackedStatus struct {
	Kind          string
	Owner         string
	Name          string
	Path          string
	Local         string
	PinnedVersion *int
	LatestVersion *int
	Remote        string
	Hint          string
}

type RunStatusSummary struct {
	Total   int
	Running int
	Stopped int
}

type StatusSummary struct {
	Modified    int
	Behind      int
	PinOutdated int
	Clean       int
}

type PullResult struct {
	Status    string
	Resources []PullResourceChange
	Manifest  *Manifest
}

type PullResourceChange struct {
	Kind string
	Name string
	From *int
	To   *int
}

type FetchResult struct {
	Status    string
	Resources []PullResourceChange
	Manifest  *Manifest
}

type PushResult struct {
	Status string
	Pushed []PushResourceChange
}

type PushResourceChange struct {
	Kind   string
	Owner  string
	Name   string
	From   *int
	To     *int
	Reason string
}

func NewService(client RemoteWorkspaceClient, fs FileMaterializer, git GitRepo) *Service {
	return &Service{client: client, fs: fs, git: git}
}

// --- Clone ---

type teamEntry struct {
	id         string
	owner      string
	name       string
	version    int
	projection *TeamProjection
}

type agentEntry struct {
	teamEntry
	localBase string
}

func (s *Service) Clone(base, owner, name string) (*Manifest, error) {
	resolved, err := s.client.ResolveTeam(owner, name)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s: %w", owner, name, err)
	}
	return s.cloneResolved(base, &resolved.Root, resolved.Resources)
}

func (s *Service) CloneVersion(base, owner, name string, version int) (*Manifest, error) {
	resolved, err := s.client.ResolveTeamVersion(owner, name, version)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s@%d: %w", owner, name, version, err)
	}
	return s.cloneResolved(base, &resolved.Root, resolved.Resources)
}

func (s *Service) cloneResolved(base string, root *remoteapi.ResolvedResource, resources []remoteapi.ResolvedResource) (*Manifest, error) {
	resourceMap := buildResourceMap(resources)
	manifest, err := s.materializeResolvedTeam(base, root, resourceMap, nil)
	if err != nil {
		return nil, err
	}
	manifest.ClonedAt = time.Now().UTC()
	if err := storemanifest.Save(s.fs, base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) materializeResolvedTeam(base string, root *remoteapi.ResolvedResource, resourceMap map[string]*remoteapi.ResolvedResource, previous *Manifest) (*Manifest, error) {
	rootProjection, err := remoteapi.TeamProjectionFromResolved(root)
	if err != nil {
		return nil, err
	}
	teams, agents, err := s.collectResolvedEntries(root.OwnerName, root.Version, rootProjection, resourceMap)
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, &domain.Fault{
			Kind: domain.KindWorkingCopyIncomplete,
			Subject: map[string]string{
				"detail": "team " + root.OwnerName + "/" + rootProjection.Name + " has no runnable agents (unknown agent type " + rootProjection.AgentType + ")",
			},
		}
	}
	localDirs := assignLocalDirs(agents, previous)

	tracked := make([]TrackedResource, 0, len(teams))
	storedTeams := make([]StoredTeamState, 0, len(teams))
	generated := []string{}

	for _, team := range teams {
		localDir := localDirs[team.id]
		storedTeams = append(storedTeams, StoredTeamState{
			Owner:      team.owner,
			Name:       team.name,
			Version:    team.version,
			LocalDir:   localDir,
			Projection: *team.projection,
		})
		tracked = append(tracked, TrackedResource{
			Kind:          string(remoteapi.KindTeam),
			AgentType:     team.projection.AgentType,
			Owner:         team.owner,
			Name:          team.name,
			LocalPath:     teamTrackedPath(team.owner, team.name),
			RemoteVersion: intPtr(team.version),
			Editable:      true,
		})
	}

	allKeys := make([]string, 0, len(agents))
	agentsByKey := make(map[string]ProtocolAgent, len(agents))
	for _, agent := range agents {
		allKeys = append(allKeys, agent.id)
		agentsByKey[agent.id] = ProtocolAgent{ID: agent.id, Owner: agent.owner, Name: agent.name}
	}

	writer := NewWriter(s.fs, s.git, resourceMap)
	for _, agent := range agents {
		agent.localBase = localDirs[agent.id]
		agentBase := filepath.Join(base, filepath.FromSlash(agent.localBase))
		if err := writer.MaterializeAgent(agentBase, agent.projection); err != nil {
			return nil, fmt.Errorf("materialize agent %s: %w", agent.id, err)
		}

		relations := buildPeerRelations(agent.id, allKeys)
		self := agentsByKey[agent.id]
		protocol := BuildAgentFacingTeamProtocol(rootProjection.Name, self, relations, agentsByKey)
		protocolPath := filepath.Join(agentBase, ".clier", TeamProtocolFileName())
		if err := s.fs.EnsureFile(protocolPath, []byte(protocol)); err != nil {
			return nil, fmt.Errorf("write protocol for %s: %w", agent.id, err)
		}

		profile, err := domain.ProfileFor(agent.projection.AgentType)
		if err != nil {
			return nil, err
		}
		generated = append(generated,
			filepath.ToSlash(filepath.Join(agent.localBase, ".clier", "work-log-protocol.md")),
			filepath.ToSlash(filepath.Join(agent.localBase, ".clier", TeamProtocolFileName())),
		)
		appendAgentTrackedResources(&tracked, &generated, agent.projection, agent.localBase, profile)
	}

	manifest := &Manifest{
		Kind:             string(remoteapi.KindTeam),
		Owner:            root.OwnerName,
		Name:             rootProjection.Name,
		Teams:            storedTeams,
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
	}
	if err := s.populateBaseHashes(base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) collectResolvedEntries(rootOwner string, rootVersion int, rootProjection *TeamProjection, resourceMap map[string]*remoteapi.ResolvedResource) ([]teamEntry, []agentEntry, error) {
	var teams []teamEntry
	var agents []agentEntry

	var walk func(owner string, version int, projection *TeamProjection) error
	walk = func(owner string, version int, projection *TeamProjection) error {
		entry := teamEntry{
			id:         ResourceID(owner, projection.Name),
			owner:      owner,
			name:       projection.Name,
			version:    version,
			projection: projection,
		}
		teams = append(teams, entry)
		runnable, err := validateProjectionAgentType(projection.AgentType)
		if err != nil {
			return err
		}
		if runnable {
			agents = append(agents, agentEntry{
				teamEntry: entry,
				localBase: AgentWorkspaceLocalPath(owner, projection.Name),
			})
		}
		for _, child := range projection.Children {
			childResource, ok := resourceMap[teamKey(child.Owner, child.Name)]
			if !ok {
				return internalFault("resolve child team %s/%s: not found in resolve response", child.Owner, child.Name)
			}
			childProjection, err := remoteapi.TeamProjectionFromResolved(childResource)
			if err != nil {
				return err
			}
			if err := walk(child.Owner, childResource.Version, childProjection); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(rootOwner, rootVersion, rootProjection); err != nil {
		return nil, nil, err
	}
	return teams, agents, nil
}

// --- Pull ---

func (s *Service) Pull(base string, force bool) (*PullResult, error) {
	manifest, err := storemanifest.Load(s.fs, base)
	if err != nil {
		return nil, err
	}
	pulled, err := s.pullTarget(base, manifest, force)
	if err != nil {
		return nil, err
	}
	changes := diffTrackedResourceVersions(manifest, pulled)
	if err := storemanifest.Save(s.fs, base, pulled); err != nil {
		return nil, err
	}
	status := PullStatusPulled
	if len(changes) == 0 {
		status = PullStatusAlreadyUpToDate
	}
	return &PullResult{
		Status:    status,
		Resources: changes,
		Manifest:  pulled,
	}, nil
}

func (s *Service) Fetch(base string) (*FetchResult, error) {
	manifest, err := storemanifest.Load(s.fs, base)
	if err != nil {
		return nil, err
	}
	preview, err := s.fetchTarget(manifest)
	if err != nil {
		return nil, err
	}
	changes := diffTrackedResourceVersions(manifest, preview)
	status := FetchStatusUpdatesAvailable
	if len(changes) == 0 {
		status = PullStatusAlreadyUpToDate
	}
	return &FetchResult{
		Status:    status,
		Resources: changes,
		Manifest:  preview,
	}, nil
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
			return nil, &domain.Fault{
				Kind:    domain.KindPullBlockedDirty,
				Subject: map[string]string{"modified": strings.Join(paths, ", ")},
			}
		}
	}

	// Use ResolveTeam for a full refresh.
	resolved, err := s.client.ResolveTeam(manifest.Owner, manifest.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s: %w", manifest.Owner, manifest.Name, err)
	}
	resourceMap := buildResourceMap(resolved.Resources)
	// Also add root to map for child lookups.
	rootKey := teamKey(resolved.Root.OwnerName, resolved.Root.Name)
	resourceMap[rootKey] = &resolved.Root
	pulled, err := s.materializeResolvedTeam(base, &resolved.Root, resourceMap, manifest)
	if err != nil {
		return nil, err
	}
	pulled.ClonedAt = manifest.ClonedAt
	if err := s.removeStaleManagedFiles(base, manifest, pulled); err != nil {
		return nil, err
	}
	return pulled, nil
}

func (s *Service) fetchTarget(manifest *Manifest) (*Manifest, error) {
	resolved, err := s.client.ResolveTeam(manifest.Owner, manifest.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s: %w", manifest.Owner, manifest.Name, err)
	}
	return s.previewResolvedManifest(&resolved.Root, resolved.Resources, manifest)
}

// --- Status ---

func (s *Service) Status(base, runsDir string) (*Status, error) {
	manifest, err := storemanifest.Load(s.fs, base)
	if err != nil {
		return nil, err
	}
	tracked, modifiedCount, err := s.trackedStatuses(base, manifest)
	if err != nil {
		return nil, err
	}
	runs, err := s.runSummary(base, runsDir)
	if err != nil {
		return nil, err
	}
	local := LocalStatusClean
	if modifiedCount > 0 {
		local = LocalStatusModified
	}
	latestTeamVersions, err := s.latestTeamTrackedVersions(manifest)
	if err != nil {
		return nil, err
	}
	pullHint := PullHint(manifest.Owner, manifest.Name)
	// Check remote versions — deduplicate by owner/name to avoid redundant API calls.
	type remoteVersion struct {
		latest int
		err    error
	}
	remoteCache := map[string]*remoteVersion{}
	trackedIndex := make(map[string]int, len(tracked))
	for i, tr := range tracked {
		trackedIndex[tr.Path] = i
	}
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
			return nil, fmt.Errorf("get resource %s/%s: %w", tr.Owner, tr.Name, rv.err)
		}
		idx, ok := trackedIndex[manifest.TrackedResources[i].LocalPath]
		if !ok {
			continue
		}
		pinned := *tr.RemoteVersion
		tracked[idx].PinnedVersion = &pinned
		tracked[idx].LatestVersion = &rv.latest
		desired, hasDesired := latestTeamVersions[manifest.TrackedResources[i].LocalPath]
		switch {
		case hasDesired && pinned < desired:
			tracked[idx].Remote = RemoteStatusBehind
			tracked[idx].Hint = pullHint
		case hasDesired && pinned == desired && pinned < rv.latest:
			tracked[idx].Remote = RemoteStatusPinOutdated
			tracked[idx].Hint = PinOutdatedHint()
		case pinned < rv.latest:
			tracked[idx].Remote = RemoteStatusBehind
			tracked[idx].Hint = pullHint
		default:
			tracked[idx].Remote = RemoteStatusUpToDate
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
		Summary: summarizeTrackedStatuses(tracked),
		Tracked: tracked,
		Runs:    runs,
	}

	return status, nil
}

func (s *Service) latestTeamTrackedVersions(manifest *Manifest) (map[string]int, error) {
	root, err := s.client.GetResource(manifest.Owner, manifest.Name)
	if err != nil {
		return nil, err
	}
	resolved, err := s.client.ResolveTeamVersion(manifest.Owner, manifest.Name, root.Metadata.LatestVersion)
	if err != nil {
		return nil, err
	}

	preview, err := s.previewResolvedManifest(&resolved.Root, resolved.Resources, manifest)
	if err != nil {
		return nil, err
	}

	versions := make(map[string]int, len(preview.TrackedResources))
	for _, resource := range preview.TrackedResources {
		if resource.RemoteVersion == nil {
			continue
		}
		versions[resource.LocalPath] = *resource.RemoteVersion
	}
	return versions, nil
}

// --- Push ---

func (s *Service) Push(base string) (*PushResult, error) {
	manifest, err := storemanifest.Load(s.fs, base)
	if err != nil {
		return nil, err
	}
	modified, err := s.ModifiedTrackedResources(base)
	if err != nil {
		return nil, err
	}
	if len(modified) == 0 {
		return &PushResult{Status: PushStatusNoChanges, Pushed: []PushResourceChange{}}, nil
	}

	originallyModified := make(map[string]bool, len(modified))
	for _, resource := range modified {
		originallyModified[resource.LocalPath] = true
	}

	pushed := make([]PushResourceChange, 0, len(modified))
	targetName := manifest.Name
	pushedPaths := map[string]bool{}

	for _, resource := range modified {
		if !resource.Editable || remoteapi.ResourceKind(resource.Kind) == remoteapi.KindTeam {
			continue
		}
		updated, err := s.pushTrackedResource(base, manifest, resource)
		if err != nil {
			return nil, err
		}
		pushedPaths[resource.LocalPath] = true
		if updated != nil {
			if resource.LocalPath == manifest.RootResource.LocalPath {
				targetName = updated.Metadata.Name
			}
			pushed = append(pushed, buildPushResourceChange(resource, updated.Metadata.LatestVersion, PushReasonLocalEdit))
			s.applyPushedResourceVersion(manifest, resource, updated.Metadata.LatestVersion)
			if err := s.recordPushedResourceState(base, manifest, resource); err != nil {
				return nil, err
			}
		}
	}

	for {
		progress := false
		for _, resource := range manifest.TrackedResources {
			if !resource.Editable || remoteapi.ResourceKind(resource.Kind) != remoteapi.KindTeam {
				continue
			}
			if pushedPaths[resource.LocalPath] {
				continue
			}
			modified, blocked, err := s.teamPushState(base, manifest, resource)
			if err != nil {
				return nil, err
			}
			if !modified || blocked {
				continue
			}
			updated, err := s.pushTrackedResource(base, manifest, resource)
			if err != nil {
				return nil, err
			}
			pushedPaths[resource.LocalPath] = true
			progress = true
			if updated != nil {
				if resource.LocalPath == manifest.RootResource.LocalPath {
					targetName = updated.Metadata.Name
				}
				reason := PushReasonRefCascade
				if originallyModified[resource.LocalPath] {
					reason = PushReasonLocalEdit
				}
				pushed = append(pushed, buildPushResourceChange(resource, updated.Metadata.LatestVersion, reason))
				s.applyPushedResourceVersion(manifest, resource, updated.Metadata.LatestVersion)
				if err := s.recordPushedResourceState(base, manifest, resource); err != nil {
					return nil, err
				}
			}
		}
		if !progress {
			break
		}
	}

	for _, resource := range manifest.TrackedResources {
		if !resource.Editable || remoteapi.ResourceKind(resource.Kind) != remoteapi.KindTeam {
			continue
		}
		modified, _, err := s.teamPushState(base, manifest, resource)
		if err != nil {
			return nil, err
		}
		if modified {
			return nil, &domain.Fault{
				Kind: domain.KindWorkspaceDirty,
				Subject: map[string]string{
					"modified": "team " + resource.Owner + "/" + resource.Name + " has unresolved local changes",
				},
			}
		}
	}

	// Update manifest name if root resource was renamed, then pull latest.
	manifest.Name = targetName
	pulled, err := s.pullTarget(base, manifest, true)
	if err != nil {
		return nil, err
	}
	if err := storemanifest.Save(s.fs, base, pulled); err != nil {
		return nil, err
	}
	return &PushResult{Status: PushStatusPushed, Pushed: pushed}, nil
}

func (s *Service) pushTrackedResource(base string, manifest *Manifest, resource TrackedResource) (*remoteapi.ResourceResponse, error) {
	kind, body, err := s.preparePushBody(base, manifest, resource)
	if err != nil {
		return nil, err
	}
	return s.pushResource(resource, kind, body)
}

func (s *Service) teamPushState(base string, manifest *Manifest, resource TrackedResource) (modified bool, blocked bool, err error) {
	sum, err := s.trackedResourceHash(base, manifest, resource)
	if err != nil {
		return false, false, err
	}
	if sum == resource.BaseHash {
		return false, false, nil
	}

	team, ok := manifest.FindTeam(resource.Owner, resource.Name)
	if !ok {
		return false, false, internalFault("team state %s/%s not found", resource.Owner, resource.Name)
	}
	for _, child := range team.Projection.Children {
		childTracked, ok := manifest.FindTrackedResource(teamTrackedPath(child.Owner, child.Name))
		if !ok || !childTracked.Editable {
			continue
		}
		childSum, err := s.trackedResourceHash(base, manifest, *childTracked)
		if err != nil {
			return false, false, err
		}
		if childSum != childTracked.BaseHash {
			return true, true, nil
		}
	}
	return true, false, nil
}

func (s *Service) applyPushedResourceVersion(manifest *Manifest, resource TrackedResource, latest int) {
	for i := range manifest.TrackedResources {
		if manifest.TrackedResources[i].Kind != resource.Kind {
			continue
		}
		if manifest.TrackedResources[i].Owner != resource.Owner || manifest.TrackedResources[i].Name != resource.Name {
			continue
		}
		manifest.TrackedResources[i].RemoteVersion = intPtr(latest)
	}

	if resource.Kind == string(remoteapi.KindTeam) {
		if manifest.RootResource.Kind == resource.Kind && manifest.RootResource.Owner == resource.Owner && manifest.RootResource.Name == resource.Name {
			manifest.RootResource.RemoteVersion = intPtr(latest)
		}
		for i := range manifest.Teams {
			if manifest.Teams[i].Owner == resource.Owner && manifest.Teams[i].Name == resource.Name {
				manifest.Teams[i].Version = latest
			}
			for j := range manifest.Teams[i].Projection.Children {
				child := &manifest.Teams[i].Projection.Children[j]
				if child.Owner == resource.Owner && child.Name == resource.Name {
					child.ChildVersion = latest
				}
			}
		}
		return
	}

	for i := range manifest.Teams {
		projection := &manifest.Teams[i].Projection
		if resource.Kind == string(remoteapi.KindInstruction) && projection.InstructionRef != nil &&
			projection.InstructionRef.Owner == resource.Owner && projection.InstructionRef.Name == resource.Name {
			projection.InstructionRef.Version = latest
		}
		if (resource.Kind == string(remoteapi.KindClaudeSettings) || resource.Kind == string(remoteapi.KindCodexSettings)) &&
			projection.SettingsRef != nil &&
			projection.SettingsRef.Owner == resource.Owner && projection.SettingsRef.Name == resource.Name {
			projection.SettingsRef.Version = latest
		}
		if resource.Kind == string(remoteapi.KindSkill) {
			for j := range projection.Skills {
				skill := &projection.Skills[j]
				if skill.Owner == resource.Owner && skill.Name == resource.Name {
					skill.Version = latest
				}
			}
		}
	}
}

func (s *Service) recordPushedResourceState(base string, manifest *Manifest, resource TrackedResource) error {
	sum, err := s.trackedResourceHash(base, manifest, resource)
	if err != nil {
		return err
	}
	for i := range manifest.TrackedResources {
		if manifest.TrackedResources[i].Kind != resource.Kind {
			continue
		}
		if manifest.TrackedResources[i].Owner != resource.Owner || manifest.TrackedResources[i].Name != resource.Name {
			continue
		}
		manifest.TrackedResources[i].BaseHash = sum
	}
	if manifest.RootResource.Kind == resource.Kind && manifest.RootResource.Owner == resource.Owner && manifest.RootResource.Name == resource.Name {
		manifest.RootResource.BaseHash = sum
	}
	return nil
}

// preparePushBody builds the request body for a single tracked resource.
func (s *Service) preparePushBody(base string, manifest *Manifest, r TrackedResource) (remoteapi.ResourceKind, any, error) {
	kind := remoteapi.ResourceKind(r.Kind)
	switch kind {
	case remoteapi.KindTeam:
		team, ok := manifest.FindTeam(r.Owner, r.Name)
		if !ok {
			return "", nil, internalFault("team state %s/%s not found", r.Owner, r.Name)
		}
		body, err := s.teamMutationFromProjection(&team.Projection)
		if err != nil {
			return "", nil, err
		}
		return kind, body, nil
	default:
		content, err := s.readContentForPush(base, manifest, kind, r)
		if err != nil {
			return "", nil, err
		}
		return kind, remoteapi.ContentWriteRequest{Name: r.Name, Content: content}, nil
	}
}

// pushResource checks the remote version and uploads the resource.
func (s *Service) pushResource(r TrackedResource, kind remoteapi.ResourceKind, body any) (*remoteapi.ResourceResponse, error) {
	current, err := s.client.GetResource(r.Owner, r.Name)
	if err != nil {
		return nil, err
	}
	if !versionsMatch(r.RemoteVersion, current.Metadata.LatestVersion) {
		return nil, &domain.Fault{
			Kind: domain.KindRemoteChanged,
			Subject: map[string]string{
				"resource_kind": r.Kind,
				"owner":         r.Owner,
				"name":          r.Name,
			},
		}
	}
	return s.client.UpdateResource(kind, r.Owner, r.Name, body)
}

// readContentForPush reads a content resource from disk, stripping the prelude for instruction kinds.
func (s *Service) readContentForPush(base string, manifest *Manifest, kind remoteapi.ResourceKind, r TrackedResource) (string, error) {
	if remoteapi.IsInstructionKind(string(kind)) {
		return s.serverInstructionContent(base, manifest, r)
	}
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(r.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", r.LocalPath, err)
	}
	return string(data), nil
}

// teamMutationFromProjection builds a TeamWriteRequest from a TeamProjection,
// resolving the team's own agent fields and its children.
func (s *Service) teamMutationFromProjection(projection *TeamProjection) (*remoteapi.TeamWriteRequest, error) {
	req := &remoteapi.TeamWriteRequest{
		Name:       projection.Name,
		Command:    projection.Command,
		GitRepoURL: projection.GitRepoURL,
	}

	// Instruction and settings refs require an agent profile to resolve the kind.
	if projection.InstructionRef != nil || projection.SettingsRef != nil {
		profile, err := domain.ProfileFor(projection.AgentType)
		if err != nil {
			return nil, fmt.Errorf("resolve agent profile: %w", err)
		}
		if projection.InstructionRef != nil {
			ref := &remoteapi.ResourceRefRequest{
				Owner:   projection.InstructionRef.Owner,
				Name:    projection.InstructionRef.Name,
				Version: projection.InstructionRef.Version,
			}
			req.SetInstructionRef(ref)
		}
		if projection.SettingsRef != nil {
			ref := &remoteapi.ResourceRefRequest{
				Owner:   projection.SettingsRef.Owner,
				Name:    projection.SettingsRef.Name,
				Version: projection.SettingsRef.Version,
			}
			req.SetSettingsRef(profile.SettingsKind, ref)
		}
	}

	// Skills
	skillRefs := make([]remoteapi.ResourceRefRequest, 0, len(projection.Skills))
	for _, skillRef := range projection.Skills {
		skillRefs = append(skillRefs, remoteapi.ResourceRefRequest{
			Owner:   skillRef.Owner,
			Name:    skillRef.Name,
			Version: skillRef.Version,
		})
	}
	req.Skills = skillRefs

	// Children
	children := make([]remoteapi.ChildRefRequest, 0, len(projection.Children))
	for _, child := range projection.Children {
		children = append(children, remoteapi.ChildRefRequest{
			Owner:        child.Owner,
			Name:         child.Name,
			ChildVersion: child.ChildVersion,
		})
	}
	req.Children = children

	return req, nil
}

// --- Modified / Tracked ---

func (s *Service) ModifiedTrackedResources(base string) ([]TrackedResource, error) {
	manifest, err := storemanifest.Load(s.fs, base)
	if err != nil {
		return nil, err
	}

	var modified []TrackedResource
	for _, resource := range manifest.TrackedResources {
		sum, err := s.trackedResourceHash(base, manifest, resource)
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
		sum, err := s.trackedResourceHash(base, manifest, resource)
		if err != nil {
			return nil, 0, err
		}
		local := LocalStatusClean
		if sum != resource.BaseHash {
			local = LocalStatusModified
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

// runSummary delegates to apprun.ListPlans (single scan policy) and
// counts plans whose WorkingCopyPath matches the given base.
func (s *Service) runSummary(base, runsDir string) (RunStatusSummary, error) {
	if runsDir == "" {
		return RunStatusSummary{}, nil
	}
	plans, err := apprun.NewRepository(runsDir).ListForWorkingCopy(base)
	if err != nil {
		return RunStatusSummary{}, err
	}
	var summary RunStatusSummary
	for _, plan := range plans {
		summary.Total++
		if plan.Status == apprun.StatusRunning {
			summary.Running++
		} else {
			summary.Stopped++
		}
	}
	return summary, nil
}

// serverInstructionContent reads the local instruction file and strips the
// agent-specific prelude before sending to the server.
func (s *Service) serverInstructionContent(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
	}
	content := string(data)

	// Try to find the agent type by looking up the agent's team projection.
	agentType := resource.AgentType
	if agentType == "" {
		agentType = "claude"
	}
	if _, ok := manifest.AgentForLocalPath(resource.LocalPath); !ok {
		return "", internalFault("derive agent id from %s: no matching local dir in state", resource.LocalPath)
	}
	return StripInstructionPrelude(agentType, content), nil
}

// --- Helpers ---

// buildResourceMap indexes resolved resources by owner/name for O(1) lookup.
func buildResourceMap(resources []remoteapi.ResolvedResource) map[string]*remoteapi.ResolvedResource {
	m := make(map[string]*remoteapi.ResolvedResource, len(resources))
	for i := range resources {
		r := &resources[i]
		m[teamKey(r.OwnerName, r.Name)] = r
	}
	return m
}

// appendAgentTrackedResources adds tracked resources and generated files for
// a team's agent role to the tracked/generated slices.
func appendAgentTrackedResources(tracked *[]TrackedResource, generated *[]string, projection *TeamProjection, localBase string, profile domain.AgentProfile) {
	agentType := projection.AgentType

	if projection.InstructionRef != nil {
		*tracked = append(*tracked, TrackedResource{
			Kind:          profile.InstructionKind,
			AgentType:     agentType,
			Owner:         projection.InstructionRef.Owner,
			Name:          projection.InstructionRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(localBase, profile.InstructionFile)),
			RemoteVersion: intPtr(projection.InstructionRef.Version),
			Editable:      true,
		})
	} else if generated != nil {
		*generated = append(*generated, filepath.ToSlash(filepath.Join(localBase, profile.InstructionFile)))
	}

	if projection.SettingsRef != nil {
		*tracked = append(*tracked, TrackedResource{
			Kind:          profile.SettingsKind,
			AgentType:     agentType,
			Owner:         projection.SettingsRef.Owner,
			Name:          projection.SettingsRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(localBase, profile.SettingsDir, profile.SettingsFile)),
			RemoteVersion: intPtr(projection.SettingsRef.Version),
			Editable:      true,
		})
	}

	for _, skillRef := range projection.Skills {
		*tracked = append(*tracked, TrackedResource{
			Kind:          string(remoteapi.KindSkill),
			AgentType:     agentType,
			Owner:         skillRef.Owner,
			Name:          skillRef.Name,
			LocalPath:     SkillLocalPath(path.Join(localBase, profile.SettingsDir, profile.SkillsDir), skillRef.Owner, skillRef.Name),
			RemoteVersion: intPtr(skillRef.Version),
			Editable:      true,
		})
	}
}

// buildPeerRelations builds TeamRelations where all other keys are peers (workers).
// In the new flat model, children are peers — no leader/worker hierarchy.
func buildPeerRelations(selfKey string, allKeys []string) domain.TeamRelations {
	workers := make([]string, 0, len(allKeys)-1)
	for _, key := range allKeys {
		if key != selfKey {
			workers = append(workers, key)
		}
	}
	return domain.TeamRelations{
		Leaders: []string{},
		Workers: workers,
	}
}

// teamKey returns the unique identifier for a team (owner/name).
func teamKey(owner, name string) string {
	return ResourceID(owner, name)
}

func (s *Service) removeStaleManagedFiles(base string, previous, next *Manifest) error {
	keep := make(map[string]struct{}, len(next.TrackedResources)+len(next.GeneratedFiles))
	for _, resource := range next.TrackedResources {
		keep[resource.LocalPath] = struct{}{}
	}
	for _, path := range next.GeneratedFiles {
		keep[path] = struct{}{}
	}

	for _, resource := range previous.TrackedResources {
		if _, ok := keep[resource.LocalPath]; ok {
			continue
		}
		if err := s.fs.RemoveAll(filepath.Join(base, filepath.FromSlash(resource.LocalPath))); err != nil {
			return fmt.Errorf("remove stale tracked file %s: %w", resource.LocalPath, err)
		}
	}
	for _, path := range previous.GeneratedFiles {
		if _, ok := keep[path]; ok {
			continue
		}
		if err := s.fs.RemoveAll(filepath.Join(base, filepath.FromSlash(path))); err != nil {
			return fmt.Errorf("remove stale generated file %s: %w", path, err)
		}
	}
	return nil
}

func (s *Service) populateBaseHashes(base string, manifest *Manifest) error {
	for i := range manifest.TrackedResources {
		sum, err := s.trackedResourceHash(base, manifest, manifest.TrackedResources[i])
		if err != nil {
			return err
		}
		manifest.TrackedResources[i].BaseHash = sum
	}
	if len(manifest.TrackedResources) > 0 {
		manifest.RootResource = manifest.TrackedResources[0]
	}
	return nil
}

func intPtr(v int) *int {
	return &v
}

func summarizeTrackedStatuses(tracked []TrackedStatus) StatusSummary {
	var summary StatusSummary
	for _, resource := range tracked {
		switch {
		case resource.Local == LocalStatusModified:
			summary.Modified++
		case resource.Remote == RemoteStatusBehind:
			summary.Behind++
		case resource.Remote == RemoteStatusPinOutdated:
			summary.PinOutdated++
		default:
			summary.Clean++
		}
	}
	return summary
}

func buildPushResourceChange(resource TrackedResource, latest int, reason string) PushResourceChange {
	return PushResourceChange{
		Kind:   resource.Kind,
		Owner:  resource.Owner,
		Name:   resource.Name,
		From:   trackedRemoteVersion(resource, true),
		To:     intPtr(latest),
		Reason: reason,
	}
}

func (s *Service) previewResolvedManifest(root *remoteapi.ResolvedResource, resources []remoteapi.ResolvedResource, previous *Manifest) (*Manifest, error) {
	resourceMap := buildResourceMap(resources)
	rootProjection, err := remoteapi.TeamProjectionFromResolved(root)
	if err != nil {
		return nil, err
	}
	teams, agents, err := s.collectResolvedEntries(root.OwnerName, root.Version, rootProjection, resourceMap)
	if err != nil {
		return nil, err
	}
	localDirs := assignLocalDirs(agents, previous)

	tracked := make([]TrackedResource, 0, len(teams))
	storedTeams := make([]StoredTeamState, 0, len(teams))
	for _, team := range teams {
		localDir := localDirs[team.id]
		storedTeams = append(storedTeams, StoredTeamState{
			Owner:      team.owner,
			Name:       team.name,
			Version:    team.version,
			LocalDir:   localDir,
			Projection: *team.projection,
		})
		tracked = append(tracked, TrackedResource{
			Kind:          string(remoteapi.KindTeam),
			AgentType:     team.projection.AgentType,
			Owner:         team.owner,
			Name:          team.name,
			LocalPath:     teamTrackedPath(team.owner, team.name),
			RemoteVersion: intPtr(team.version),
			Editable:      true,
		})
	}
	for _, agent := range agents {
		profile, err := domain.ProfileFor(agent.projection.AgentType)
		if err != nil {
			return nil, err
		}
		appendAgentTrackedResources(&tracked, nil, agent.projection, localDirs[agent.id], profile)
	}

	preview := &Manifest{
		Kind:             string(remoteapi.KindTeam),
		Owner:            root.OwnerName,
		Name:             rootProjection.Name,
		RootResource:     tracked[0],
		Teams:            storedTeams,
		TrackedResources: tracked,
	}
	if previous != nil {
		preview.ClonedAt = previous.ClonedAt
	}
	return preview, nil
}

func diffTrackedResourceVersions(previous, next *Manifest) []PullResourceChange {
	prevByPath := make(map[string]TrackedResource, len(previous.TrackedResources))
	pathSet := make(map[string]struct{}, len(previous.TrackedResources)+len(next.TrackedResources))
	for _, resource := range previous.TrackedResources {
		prevByPath[resource.LocalPath] = resource
		pathSet[resource.LocalPath] = struct{}{}
	}

	nextByPath := make(map[string]TrackedResource, len(next.TrackedResources))
	for _, resource := range next.TrackedResources {
		nextByPath[resource.LocalPath] = resource
		pathSet[resource.LocalPath] = struct{}{}
	}

	paths := make([]string, 0, len(pathSet))
	for path := range pathSet {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	changes := make([]PullResourceChange, 0)
	for _, path := range paths {
		prev, prevOK := prevByPath[path]
		curr, currOK := nextByPath[path]

		prevVersion := trackedRemoteVersion(prev, prevOK)
		currVersion := trackedRemoteVersion(curr, currOK)
		if versionsEqual(prevVersion, currVersion) {
			continue
		}

		changes = append(changes, PullResourceChange{
			Kind: resourceKindForChange(prev, prevOK, curr),
			Name: resourceNameForChange(prev, prevOK, curr),
			From: prevVersion,
			To:   currVersion,
		})
	}

	return changes
}

func trackedRemoteVersion(resource TrackedResource, ok bool) *int {
	if !ok || resource.RemoteVersion == nil {
		return nil
	}
	return intPtr(*resource.RemoteVersion)
}

func resourceKindForChange(previous TrackedResource, previousOK bool, next TrackedResource) string {
	if previousOK {
		return previous.Kind
	}
	return next.Kind
}

func resourceNameForChange(previous TrackedResource, previousOK bool, next TrackedResource) string {
	if previousOK {
		return previous.Name
	}
	return next.Name
}

func versionsEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
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

func (s *Service) trackedResourceHash(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	if remoteapi.ResourceKind(resource.Kind) == remoteapi.KindTeam {
		if manifest == nil {
			return "", internalFault("team hash requires manifest state")
		}
		team, ok := manifest.FindTeam(resource.Owner, resource.Name)
		if !ok {
			return "", internalFault("team state %s/%s not found", resource.Owner, resource.Name)
		}
		data, err := MarshalTeamProjection(team.Projection)
		if err != nil {
			return "", fmt.Errorf("marshal team projection %s/%s: %w", resource.Owner, resource.Name, err)
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), nil
	}
	return s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
}

func teamTrackedPath(owner, name string) string {
	return ".clier/" + ResourceDirName(owner, name) + ".team"
}

func assignLocalDirs(agents []agentEntry, previous *Manifest) map[string]string {
	assigned := make(map[string]string, len(agents))
	taken := map[string]bool{}

	if previous != nil {
		for _, team := range previous.Teams {
			if team.LocalDir != "" {
				taken[team.LocalDir] = true
			}
		}
	}

	for _, agent := range agents {
		if previous != nil {
			if prior, ok := previous.FindTeam(agent.owner, agent.name); ok && prior.LocalDir != "" {
				assigned[agent.id] = prior.LocalDir
				continue
			}
		}
		base := AgentWorkspaceLocalPath(agent.owner, agent.name)
		candidate := base
		for i := 2; taken[candidate]; i++ {
			candidate = fmt.Sprintf("%s__%d", base, i)
		}
		taken[candidate] = true
		assigned[agent.id] = candidate
	}

	return assigned
}

func versionsMatch(expected *int, actual int) bool {
	if expected == nil {
		return false
	}
	return *expected == actual
}
