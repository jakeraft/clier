package workspace

import (
	"fmt"
	"path/filepath"

	domainworkspace "github.com/jakeraft/clier/internal/domain/workspace"
	"github.com/jakeraft/clier/internal/store/workspacecodec"
)

type ResourceRefProjection = domainworkspace.ResourceRef
type TeamProjection = domainworkspace.TeamProjection
type ChildProjection = domainworkspace.Child

const TeamProjectionFile = "team.json"

func TeamProjectionPath(base string) string {
	return filepath.Join(base, ".clier", TeamProjectionFile)
}

func TeamProjectionLocalPath() string {
	return filepath.ToSlash(filepath.Join(".clier", "team.json"))
}

func WriteTeamProjection(fs FileMaterializer, path string, projection *TeamProjection) error {
	data, err := workspacecodec.MarshalIndent(*projection)
	if err != nil {
		return fmt.Errorf("marshal projection: %w", err)
	}
	if err := fs.EnsureFile(path, data); err != nil {
		return fmt.Errorf("write projection: %w", err)
	}
	return nil
}

func LoadTeamProjection(fs FileMaterializer, path string) (*TeamProjection, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read projection: %w", err)
	}
	projection, err := workspacecodec.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal projection: %w", err)
	}
	return &projection, nil
}

func MarshalTeamProjection(projection TeamProjection) ([]byte, error) {
	return workspacecodec.Marshal(projection)
}
