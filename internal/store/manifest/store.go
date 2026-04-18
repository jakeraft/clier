package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jakeraft/clier/internal/domain"
	domainworkspace "github.com/jakeraft/clier/internal/domain/workspace"
	"github.com/jakeraft/clier/internal/store/workspacecodec"
)

const (
	FileName      = "state.json"
	CurrentFormat = 1
)

type fileStore interface {
	EnsureFile(path string, content []byte) error
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
}

type record struct {
	Format           int               `json:"format"`
	Kind             string            `json:"kind"`
	Owner            string            `json:"owner"`
	Name             string            `json:"name"`
	ClonedAt         time.Time         `json:"cloned_at"`
	FirstRunAt       *time.Time        `json:"first_run_at,omitempty"`
	RootResource     trackedRecord     `json:"root_resource"`
	Teams            []teamStateRecord `json:"teams,omitempty"`
	TrackedResources []trackedRecord   `json:"tracked_resources,omitempty"`
	GeneratedFiles   []string          `json:"generated_files,omitempty"`
}

type teamStateRecord struct {
	Owner      string                              `json:"owner"`
	Name       string                              `json:"name"`
	Version    int                                 `json:"version"`
	LocalDir   string                              `json:"local_dir,omitempty"`
	Projection workspacecodec.TeamProjectionRecord `json:"projection"`
}

type trackedRecord struct {
	Kind          string `json:"kind"`
	AgentType     string `json:"agent_type,omitempty"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	LocalPath     string `json:"local_path"`
	RemoteVersion *int   `json:"remote_version,omitempty"`
	BaseHash      string `json:"base_hash,omitempty"`
	Editable      bool   `json:"editable"`
}

func Path(base string) string {
	return filepath.Join(base, ".clier", FileName)
}

func FindPath(fs fileStore, base string) (string, error) {
	path := Path(base)
	if _, err := fs.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat working-copy manifest: %w", err)
	}
	return "", os.ErrNotExist
}

func Save(fs fileStore, base string, manifest *domainworkspace.Manifest) error {
	rec := recordFromDomain(manifest)
	rec.Format = CurrentFormat
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := fs.EnsureFile(Path(base), data); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func Load(fs fileStore, base string) (*domainworkspace.Manifest, error) {
	path, err := FindPath(fs, base)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read manifest: %w", err)
		}
		return nil, err
	}
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var rec record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	if rec.Format > CurrentFormat {
		return nil, &domain.Fault{
			Kind: domain.KindManifestIncompatible,
			Subject: map[string]string{
				"got":      strconv.Itoa(rec.Format),
				"expected": strconv.Itoa(CurrentFormat),
				"hint":     "upgrade clier",
			},
		}
	}
	if rec.Format < CurrentFormat {
		return nil, &domain.Fault{
			Kind: domain.KindManifestIncompatible,
			Subject: map[string]string{
				"got":      strconv.Itoa(rec.Format),
				"expected": strconv.Itoa(CurrentFormat),
				"hint":     "re-clone with 'clier clone'",
			},
		}
	}
	return rec.toDomain(), nil
}

func recordFromDomain(manifest *domainworkspace.Manifest) record {
	teams := make([]teamStateRecord, 0, len(manifest.Teams))
	for _, team := range manifest.Teams {
		teams = append(teams, teamStateRecord{
			Owner:      team.Owner,
			Name:       team.Name,
			Version:    team.Version,
			LocalDir:   team.LocalDir,
			Projection: workspacecodec.RecordFromDomain(team.Projection),
		})
	}
	tracked := make([]trackedRecord, 0, len(manifest.TrackedResources))
	for _, resource := range manifest.TrackedResources {
		tracked = append(tracked, trackedRecordFromDomain(resource))
	}
	generated := append([]string(nil), manifest.GeneratedFiles...)
	return record{
		Kind:             manifest.Kind,
		Owner:            manifest.Owner,
		Name:             manifest.Name,
		ClonedAt:         manifest.ClonedAt,
		FirstRunAt:       manifest.FirstRunAt,
		RootResource:     trackedRecordFromDomain(manifest.RootResource),
		Teams:            teams,
		TrackedResources: tracked,
		GeneratedFiles:   generated,
	}
}

func (r record) toDomain() *domainworkspace.Manifest {
	teams := make([]domainworkspace.TeamState, 0, len(r.Teams))
	for _, team := range r.Teams {
		teams = append(teams, domainworkspace.TeamState{
			Owner:      team.Owner,
			Name:       team.Name,
			Version:    team.Version,
			LocalDir:   team.LocalDir,
			Projection: team.Projection.ToDomain(),
		})
	}
	tracked := make([]domainworkspace.TrackedResource, 0, len(r.TrackedResources))
	for _, resource := range r.TrackedResources {
		tracked = append(tracked, resource.toDomain())
	}
	generated := append([]string(nil), r.GeneratedFiles...)
	return &domainworkspace.Manifest{
		Kind:             r.Kind,
		Owner:            r.Owner,
		Name:             r.Name,
		ClonedAt:         r.ClonedAt,
		FirstRunAt:       r.FirstRunAt,
		RootResource:     r.RootResource.toDomain(),
		Teams:            teams,
		TrackedResources: tracked,
		GeneratedFiles:   generated,
	}
}

func trackedRecordFromDomain(resource domainworkspace.TrackedResource) trackedRecord {
	return trackedRecord{
		Kind:          resource.Kind,
		AgentType:     resource.AgentType,
		Owner:         resource.Owner,
		Name:          resource.Name,
		LocalPath:     resource.LocalPath,
		RemoteVersion: resource.RemoteVersion,
		BaseHash:      resource.BaseHash,
		Editable:      resource.Editable,
	}
}

func (r trackedRecord) toDomain() domainworkspace.TrackedResource {
	return domainworkspace.TrackedResource{
		Kind:          r.Kind,
		AgentType:     r.AgentType,
		Owner:         r.Owner,
		Name:          r.Name,
		LocalPath:     r.LocalPath,
		RemoteVersion: r.RemoteVersion,
		BaseHash:      r.BaseHash,
		Editable:      r.Editable,
	}
}
