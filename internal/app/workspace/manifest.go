package workspace

import (
	domainworkspace "github.com/jakeraft/clier/internal/domain/workspace"
	storemanifest "github.com/jakeraft/clier/internal/store/manifest"
)

type Manifest = domainworkspace.Manifest
type StoredTeamState = domainworkspace.TeamState
type TrackedResource = domainworkspace.TrackedResource

func ManifestPath(base string) string {
	return storemanifest.Path(base)
}

func LoadManifest(fs FileMaterializer, base string) (*Manifest, error) {
	return storemanifest.Load(fs, base)
}

func SaveManifest(fs FileMaterializer, base string, manifest *Manifest) error {
	return storemanifest.Save(fs, base, manifest)
}
