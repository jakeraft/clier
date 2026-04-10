package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
