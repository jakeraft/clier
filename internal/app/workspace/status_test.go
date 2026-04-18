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
		case "/api/v1/orgs/org/resources/root":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "root", OwnerName: "org", LatestVersion: 2},
			})
		case "/api/v1/orgs/org/resources/alpha":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "alpha", OwnerName: "org", LatestVersion: 22},
			})
		case "/api/v1/orgs/org/resources/zeta":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "zeta", OwnerName: "org", LatestVersion: 11},
			})
		case "/api/v1/orgs/org/teams/root/versions/2/resolve":
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "root",
					Version:   2,
					Snapshot:  []byte(`{"agent_type":"manager","children":[{"owner":"org","name":"alpha","version":22},{"owner":"org","name":"zeta","version":11}]}`),
				},
				Resources: []api.ResolvedResource{
					{
						Kind:      string(api.KindTeam),
						OwnerName: "org",
						Name:      "alpha",
						Version:   22,
						Snapshot:  []byte(`{"agent_type":"manager"}`),
					},
					{
						Kind:      string(api.KindTeam),
						OwnerName: "org",
						Name:      "zeta",
						Version:   11,
						Snapshot:  []byte(`{"agent_type":"manager"}`),
					},
				},
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
	if status.Summary != (StatusSummary{Behind: 2}) {
		t.Fatalf("summary = %+v, want behind=2", status.Summary)
	}
}

func TestStatus_DistinguishesBehindFromPinOutdated(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	rootProjection := TeamProjection{
		Name: "root",
		Children: []ChildProjection{
			{Owner: "org", Name: "alpha", ChildVersion: 2},
			{Owner: "org", Name: "zeta", ChildVersion: 1},
		},
	}
	alphaProjection := TeamProjection{Name: "alpha"}
	zetaProjection := TeamProjection{Name: "zeta"}

	manifest := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "root",
		ClonedAt: time.Unix(0, 0).UTC(),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "root",
			LocalPath:     teamTrackedPath("org", "root"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, rootProjection),
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "root", Version: 1, Projection: rootProjection},
			{Owner: "org", Name: "alpha", Version: 2, Projection: alphaProjection},
			{Owner: "org", Name: "zeta", Version: 1, Projection: zetaProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "root",
				LocalPath:     teamTrackedPath("org", "root"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, rootProjection),
			},
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "alpha",
				LocalPath:     teamTrackedPath("org", "alpha"),
				RemoteVersion: intPtr(2),
				BaseHash:      hashTeamProjectionForTest(t, alphaProjection),
			},
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "zeta",
				LocalPath:     teamTrackedPath("org", "zeta"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, zetaProjection),
			},
		},
	}
	if err := SaveManifest(fs, base, manifest); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/orgs/org/resources/root":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "root", OwnerName: "org", LatestVersion: 5},
			})
		case "/api/v1/orgs/org/resources/alpha":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "alpha", OwnerName: "org", LatestVersion: 3},
			})
		case "/api/v1/orgs/org/resources/zeta":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "zeta", OwnerName: "org", LatestVersion: 3},
			})
		case "/api/v1/orgs/org/teams/root/versions/5/resolve":
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "root",
					Version:   5,
					Snapshot:  []byte(`{"agent_type":"manager","children":[{"owner":"org","name":"alpha","version":3},{"owner":"org","name":"zeta","version":1}]}`),
				},
				Resources: []api.ResolvedResource{
					{
						Kind:      string(api.KindTeam),
						OwnerName: "org",
						Name:      "alpha",
						Version:   3,
						Snapshot:  []byte(`{"agent_type":"manager"}`),
					},
					{
						Kind:      string(api.KindTeam),
						OwnerName: "org",
						Name:      "zeta",
						Version:   1,
						Snapshot:  []byte(`{"agent_type":"manager"}`),
					},
				},
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

	root := gotByPath[teamTrackedPath("org", "root")]
	if root.Remote != RemoteStatusBehind {
		t.Fatalf("root remote = %q, want behind", root.Remote)
	}
	if root.Hint != PullHint("org", "root") {
		t.Fatalf("root hint = %q", root.Hint)
	}

	alpha := gotByPath[teamTrackedPath("org", "alpha")]
	if alpha.Remote != RemoteStatusBehind {
		t.Fatalf("alpha remote = %q, want behind", alpha.Remote)
	}
	if alpha.Hint != PullHint("org", "root") {
		t.Fatalf("alpha hint = %q", alpha.Hint)
	}

	zeta := gotByPath[teamTrackedPath("org", "zeta")]
	if zeta.Remote != RemoteStatusPinOutdated {
		t.Fatalf("zeta remote = %q, want pin_outdated", zeta.Remote)
	}
	if zeta.Hint != PinOutdatedHint() {
		t.Fatalf("zeta hint = %q", zeta.Hint)
	}
	if status.Summary != (StatusSummary{Behind: 2, PinOutdated: 1}) {
		t.Fatalf("summary = %+v, want behind=2 pin_outdated=1", status.Summary)
	}
}

func TestStatus_FailsWhenLatestTeamComparisonFails(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	rootProjection := TeamProjection{Name: "root"}
	manifest := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "root",
		ClonedAt: time.Unix(0, 0).UTC(),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "root",
			LocalPath:     teamTrackedPath("org", "root"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, rootProjection),
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "root", Version: 1, Projection: rootProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "root",
				LocalPath:     teamTrackedPath("org", "root"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, rootProjection),
			},
		},
	}
	if err := SaveManifest(fs, base, manifest); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/orgs/org/resources/root":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "root", OwnerName: "org", LatestVersion: 5},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""), fs, nil)
	if _, err := svc.Status(base, ""); err == nil {
		t.Fatal("Status should fail when latest team comparison cannot be resolved")
	}
}

func TestStatus_FailsWhenTrackedResourceLookupFails(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	rootProjection := TeamProjection{
		Name: "root",
		Children: []ChildProjection{
			{Owner: "org", Name: "alpha", ChildVersion: 1},
		},
	}
	alphaProjection := TeamProjection{Name: "alpha"}
	manifest := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "root",
		ClonedAt: time.Unix(0, 0).UTC(),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "root",
			LocalPath:     teamTrackedPath("org", "root"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, rootProjection),
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "root", Version: 1, Projection: rootProjection},
			{Owner: "org", Name: "alpha", Version: 1, Projection: alphaProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "root",
				LocalPath:     teamTrackedPath("org", "root"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, rootProjection),
			},
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "alpha",
				LocalPath:     teamTrackedPath("org", "alpha"),
				RemoteVersion: intPtr(1),
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
		case "/api/v1/orgs/org/resources/root":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Metadata: api.ResourceMetadata{Name: "root", OwnerName: "org", LatestVersion: 2},
			})
		case "/api/v1/orgs/org/teams/root/versions/2/resolve":
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "root",
					Version:   2,
					Snapshot:  []byte(`{"agent_type":"manager","children":[{"owner":"org","name":"alpha","version":2}]}`),
				},
				Resources: []api.ResolvedResource{
					{
						Kind:      string(api.KindTeam),
						OwnerName: "org",
						Name:      "alpha",
						Version:   2,
						Snapshot:  []byte(`{"agent_type":"manager"}`),
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""), fs, nil)
	if _, err := svc.Status(base, ""); err == nil {
		t.Fatal("Status should fail when tracked resource lookup cannot be resolved")
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
