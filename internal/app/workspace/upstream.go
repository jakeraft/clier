package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FetchUpstreamResult struct {
	Status   string `json:"status"`
	Kind     string `json:"kind"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	Version  int    `json:"version"`
	Upstream string `json:"upstream"`
}

type DiffUpstreamResult struct {
	Kind       string `json:"kind"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	Version    int    `json:"version"`
	HasChanges bool   `json:"has_changes"`
	Diff       string `json:"diff"`
}

type MergeUpstreamResult struct {
	Status    string `json:"status"`
	Kind      string `json:"kind"`
	Owner     string `json:"owner"`
	Name      string `json:"name"`
	Version   int    `json:"version"`
	LocalPath string `json:"local_path"`
}

func rootProjectionPath(base, kind string) (string, error) {
	switch kind {
	case "member":
		return MemberProjectionPath(base), nil
	case "team":
		return TeamProjectionPath(base), nil
	default:
		return "", fmt.Errorf("unsupported working-copy kind %q", kind)
	}
}

func loadFetchedUpstreamVersion(manifest *Manifest) (int, error) {
	if manifest.Upstream == nil {
		return 0, errors.New("local clone has no upstream")
	}
	if manifest.Upstream.FetchedVersion == nil {
		return 0, errors.New("no fetched upstream snapshot; run `clier fetch upstream` first")
	}
	return *manifest.Upstream.FetchedVersion, nil
}

func (s *Service) FetchUpstream(base string) (*FetchUpstreamResult, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	if manifest.Upstream == nil {
		return nil, errors.New("local clone has no upstream")
	}

	latestVersion, err := s.writeFetchedUpstreamProjection(base, manifest)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	manifest.Upstream.FetchedVersion = &latestVersion
	manifest.Upstream.FetchedAt = &now
	if err := SaveManifest(s.fs, base, manifest); err != nil {
		return nil, err
	}

	return &FetchUpstreamResult{
		Status:   "fetched",
		Kind:     manifest.Kind,
		Owner:    manifest.Owner,
		Name:     manifest.Name,
		Version:  latestVersion,
		Upstream: manifest.Upstream.Owner + "/" + manifest.Upstream.Name,
	}, nil
}

func (s *Service) DiffFetchedUpstream(base string) (*DiffUpstreamResult, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	version, err := loadFetchedUpstreamVersion(manifest)
	if err != nil {
		return nil, err
	}

	localPath, err := rootProjectionPath(base, manifest.Kind)
	if err != nil {
		return nil, err
	}
	diff, hasChanges, err := s.renderProjectionDiff(localPath, UpstreamProjectionPath(base))
	if err != nil {
		return nil, err
	}

	return &DiffUpstreamResult{
		Kind:       manifest.Kind,
		Owner:      manifest.Owner,
		Name:       manifest.Name,
		Version:    version,
		HasChanges: hasChanges,
		Diff:       diff,
	}, nil
}

func (s *Service) MergeFetchedUpstream(base string) (*MergeUpstreamResult, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	version, err := loadFetchedUpstreamVersion(manifest)
	if err != nil {
		return nil, err
	}

	modified, err := s.ModifiedTrackedResources(base)
	if err != nil {
		return nil, err
	}
	if len(modified) > 0 {
		paths := make([]string, 0, len(modified))
		for _, resource := range modified {
			paths = append(paths, resource.LocalPath)
		}
		return nil, fmt.Errorf("local changes prevent merge; push or revert first: %s", strings.Join(paths, ", "))
	}

	localPath, err := rootProjectionPath(base, manifest.Kind)
	if err != nil {
		return nil, err
	}
	data, err := s.fs.ReadFile(UpstreamProjectionPath(base))
	if err != nil {
		return nil, fmt.Errorf("read fetched upstream projection: %w", err)
	}
	if err := s.fs.EnsureFile(localPath, data); err != nil {
		return nil, fmt.Errorf("write merged projection: %w", err)
	}

	return &MergeUpstreamResult{
		Status:    "merged",
		Kind:      manifest.Kind,
		Owner:     manifest.Owner,
		Name:      manifest.Name,
		Version:   version,
		LocalPath: filepath.ToSlash(localPath),
	}, nil
}

func (s *Service) writeFetchedUpstreamProjection(base string, manifest *Manifest) (int, error) {
	upstream, err := s.client.GetResource(manifest.Upstream.Owner, manifest.Upstream.Name)
	if err != nil {
		return 0, err
	}
	if upstream.Metadata.LatestVersion == nil {
		return 0, fmt.Errorf("upstream %s %s/%s has no latest version", manifest.Upstream.Kind, manifest.Upstream.Owner, manifest.Upstream.Name)
	}

	switch manifest.Upstream.Kind {
	case "member":
		version, projection, err := s.fetchUpstreamMemberProjection(manifest.Upstream.Owner, manifest.Upstream.Name, *upstream.Metadata.LatestVersion)
		if err != nil {
			return 0, err
		}
		if err := WriteMemberProjection(s.fs, UpstreamProjectionPath(base), projection); err != nil {
			return 0, err
		}
		return version, nil
	case "team":
		version, projection, err := s.fetchUpstreamTeamProjection(manifest.Upstream.Owner, manifest.Upstream.Name, *upstream.Metadata.LatestVersion)
		if err != nil {
			return 0, err
		}
		if err := WriteTeamProjection(s.fs, UpstreamProjectionPath(base), projection); err != nil {
			return 0, err
		}
		return version, nil
	default:
		return 0, fmt.Errorf("unsupported upstream kind %q", manifest.Upstream.Kind)
	}
}

func (s *Service) fetchUpstreamMemberProjection(owner, name string, version int) (int, *MemberProjection, error) {
	vr, err := s.client.GetResourceVersion(owner, name, version)
	if err != nil {
		return 0, nil, err
	}
	var projection MemberProjection
	if err := json.Unmarshal(vr.Snapshot, &projection); err != nil {
		return 0, nil, fmt.Errorf("unmarshal upstream member projection: %w", err)
	}
	if projection.Name == "" {
		projection.Name = name
	}
	return vr.Version, &projection, nil
}

type teamSnapshotProjection struct {
	Name        string                   `json:"name,omitempty"`
	TeamMembers []teamMemberProjection   `json:"team_members"`
	Relations   []TeamRelationProjection `json:"relations,omitempty"`
}

type teamMemberProjection struct {
	MemberID      int64                 `json:"member_id"`
	MemberVersion int                   `json:"member_version"`
	Name          string                `json:"name"`
	Member        ResourceRefProjection `json:"member"`
}

func (s *Service) fetchUpstreamTeamProjection(owner, name string, version int) (int, *TeamProjection, error) {
	vr, err := s.client.GetResourceVersion(owner, name, version)
	if err != nil {
		return 0, nil, err
	}
	var raw teamSnapshotProjection
	if err := json.Unmarshal(vr.Snapshot, &raw); err != nil {
		return 0, nil, fmt.Errorf("unmarshal upstream team projection: %w", err)
	}

	projection := &TeamProjection{
		Name:      raw.Name,
		Members:   make([]TeamMemberProjection, 0, len(raw.TeamMembers)),
		Relations: append([]TeamRelationProjection(nil), raw.Relations...),
	}
	if projection.Name == "" {
		projection.Name = name
	}
	for _, member := range raw.TeamMembers {
		projection.Members = append(projection.Members, TeamMemberProjection{
			MemberID:      member.MemberID,
			MemberVersion: member.MemberVersion,
			Name:          member.Name,
			Member:        member.Member,
		})
	}
	return vr.Version, projection, nil
}

func (s *Service) renderProjectionDiff(localPath, upstreamPath string) (string, bool, error) {
	if _, err := s.fs.Stat(upstreamPath); err != nil {
		if os.IsNotExist(err) {
			return "", false, errors.New("no fetched upstream snapshot; run `clier fetch upstream` first")
		}
		return "", false, fmt.Errorf("stat fetched upstream projection: %w", err)
	}

	tempDir, err := s.fs.MkdirTemp("clier-upstream-diff-*")
	if err != nil {
		return "", false, fmt.Errorf("create temp diff dir: %w", err)
	}
	defer func() { _ = s.fs.RemoveAll(tempDir) }()

	localTempPath := filepath.Join(tempDir, "local.json")
	upstreamTempPath := filepath.Join(tempDir, "upstream.json")

	localData, err := s.fs.ReadFile(localPath)
	if err != nil {
		return "", false, fmt.Errorf("read file %s: %w", localPath, err)
	}
	if err := s.fs.EnsureFile(localTempPath, localData); err != nil {
		return "", false, fmt.Errorf("write file %s: %w", localTempPath, err)
	}
	upstreamData, err := s.fs.ReadFile(upstreamPath)
	if err != nil {
		return "", false, fmt.Errorf("read file %s: %w", upstreamPath, err)
	}
	if err := s.fs.EnsureFile(upstreamTempPath, upstreamData); err != nil {
		return "", false, fmt.Errorf("write file %s: %w", upstreamTempPath, err)
	}

	return s.git.Diff(localTempPath, upstreamTempPath)
}
