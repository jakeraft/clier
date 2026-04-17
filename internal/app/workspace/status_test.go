package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
)

func TestStatus_AssignsRemoteVersionsAfterSortingTrackedPaths(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	alphaProjection := TeamProjection{Name: "alpha", AgentType: "claude"}
	zetaProjection := TeamProjection{Name: "zeta", AgentType: "codex"}

	manifest := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "root",
		ClonedAt: time.Unix(0, 0).UTC(),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "zeta",
			LocalPath:     teamTrackedPath("org", "zeta"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, zetaProjection),
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "zeta", Version: 1, Projection: zetaProjection},
			{Owner: "org", Name: "alpha", Version: 2, Projection: alphaProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "zeta",
				LocalPath:     teamTrackedPath("org", "zeta"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, zetaProjection),
			},
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "alpha",
				LocalPath:     teamTrackedPath("org", "alpha"),
				RemoteVersion: intPtr(2),
				BaseHash:      hashTeamProjectionForTest(t, alphaProjection),
			},
		},
	}
	if err := SaveManifest(fs, base, manifest); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/orgs/org/resources/alpha":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "alpha", OwnerName: "org", LatestVersion: 22},
			})
		case "/api/v1/orgs/org/resources/zeta":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "zeta", OwnerName: "org", LatestVersion: 11},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""), fs, nil)
	status, err := svc.Status(base, "")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	gotByPath := make(map[string]TrackedStatus, len(status.Tracked))
	for _, tr := range status.Tracked {
		gotByPath[tr.Path] = tr
	}

	alpha := gotByPath[teamTrackedPath("org", "alpha")]
	if alpha.PinnedVersion == nil || *alpha.PinnedVersion != 2 {
		t.Fatalf("alpha pinned version = %v, want 2", alpha.PinnedVersion)
	}
	if alpha.LatestVersion == nil || *alpha.LatestVersion != 22 {
		t.Fatalf("alpha latest version = %v, want 22", alpha.LatestVersion)
	}

	zeta := gotByPath[teamTrackedPath("org", "zeta")]
	if zeta.PinnedVersion == nil || *zeta.PinnedVersion != 1 {
		t.Fatalf("zeta pinned version = %v, want 1", zeta.PinnedVersion)
	}
	if zeta.LatestVersion == nil || *zeta.LatestVersion != 11 {
		t.Fatalf("zeta latest version = %v, want 11", zeta.LatestVersion)
	}
}

func hashTeamProjectionForTest(t *testing.T, projection TeamProjection) string {
	t.Helper()

	data, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
