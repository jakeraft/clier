package workspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
)

func TestPreservedUpstreamState_KeepsFetchedSnapshotForSameUpstream(t *testing.T) {
	t.Parallel()

	fetchedVersion := 9
	fetchedAt := timeRef(t)
	preserved := preservedUpstreamState(
		&UpstreamMetadata{
			Kind:           "team",
			Owner:          "origin",
			Name:           "squad",
			FetchedVersion: &fetchedVersion,
			FetchedAt:      fetchedAt,
		},
		&UpstreamMetadata{
			Kind:  "team",
			Owner: "origin",
			Name:  "squad",
		},
	)

	if preserved == nil || preserved.FetchedVersion == nil || *preserved.FetchedVersion != fetchedVersion {
		t.Fatalf("preserved upstream version = %#v", preserved)
	}
	if preserved.FetchedAt == nil || !preserved.FetchedAt.Equal(*fetchedAt) {
		t.Fatalf("preserved upstream fetch time = %#v", preserved)
	}
}

func TestRenderProjectionDiff_ReturnsUnifiedDiff(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	localPath := filepath.Join(base, "local.json")
	upstreamPath := filepath.Join(base, "upstream.json")

	if err := os.WriteFile(localPath, []byte("{\n  \"name\": \"local\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write local projection: %v", err)
	}
	if err := os.WriteFile(upstreamPath, []byte("{\n  \"name\": \"upstream\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write upstream projection: %v", err)
	}

	diff, hasChanges, err := renderProjectionDiff(localPath, upstreamPath)
	if err != nil {
		t.Fatalf("renderProjectionDiff: %v", err)
	}
	if !hasChanges {
		t.Fatal("expected diff to report changes")
	}
	if !strings.Contains(diff, "local") || !strings.Contains(diff, "upstream") {
		t.Fatalf("expected diff output to mention changed content, got %q", diff)
	}
}

func TestFetchUpstreamMemberProjection_PreservesResourceNameWhenSnapshotOmitsIt(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orgs/origin/members/reviewer/versions/7" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": 7,
			"content": map[string]any{
				"agent_type": "codex",
				"command":    "codex",
			},
		})
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""))
	version, projection, err := svc.fetchUpstreamMemberProjection("origin", "reviewer", 7)
	if err != nil {
		t.Fatalf("fetchUpstreamMemberProjection: %v", err)
	}
	if version != 7 {
		t.Fatalf("version = %d, want 7", version)
	}
	if projection.Name != "reviewer" {
		t.Fatalf("projection.Name = %q, want reviewer", projection.Name)
	}
}

func TestFetchUpstreamTeamProjection_PreservesResourceNameWhenSnapshotOmitsIt(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orgs/origin/teams/dev-squad/versions/11" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": 11,
			"content": map[string]any{
				"root_team_member_id": 101,
				"team_members": []map[string]any{
					{
						"id":   101,
						"name": "lead",
						"member": map[string]any{
							"owner":   "origin",
							"name":    "lead-member",
							"version": 3,
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""))
	version, projection, err := svc.fetchUpstreamTeamProjection("origin", "dev-squad", 11)
	if err != nil {
		t.Fatalf("fetchUpstreamTeamProjection: %v", err)
	}
	if version != 11 {
		t.Fatalf("version = %d, want 11", version)
	}
	if projection.Name != "dev-squad" {
		t.Fatalf("projection.Name = %q, want dev-squad", projection.Name)
	}
}
