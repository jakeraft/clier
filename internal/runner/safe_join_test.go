package runner

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeJoinUnderRunDir covers the defense-in-depth helper that
// stops a compromised server response from writing outside
// `~/.clier/runs/<run_id>/`. Every rejected case must return an
// error; every accepted case must return a path with the runDir as
// a prefix (or runDir itself for the empty-rel case).
func TestSafeJoinUnderRunDir(t *testing.T) {
	const base = "/tmp/runs/abc"

	cases := []struct {
		name    string
		runDir  string
		rel     string
		wantErr bool
		want    string
	}{
		{name: "happy single segment", runDir: base, rel: "team.a", want: filepath.Join(base, "team.a")},
		{name: "happy nested", runDir: base, rel: "team.a/sub", want: filepath.Join(base, "team.a", "sub")},
		{name: "empty rel returns clean runDir", runDir: base, rel: "", want: filepath.Clean(base)},
		{name: "dot rel resolves to runDir itself", runDir: base, rel: ".", want: filepath.Clean(base)},
		{name: "leading dotdot rejected", runDir: base, rel: "../etc", wantErr: true},
		{name: "embedded dotdot rejected", runDir: base, rel: "team.a/../../etc", wantErr: true},
		{name: "absolute path rejected", runDir: base, rel: "/etc/passwd", wantErr: true},
		{name: "NUL byte rejected", runDir: base, rel: "team\x00/etc", wantErr: true},
		{name: "empty runDir rejected", runDir: "", rel: "team.a", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := safeJoinUnderRunDir(tc.runDir, tc.rel)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("path: got %q, want %q", got, tc.want)
			}
			if !strings.HasPrefix(got, filepath.Clean(tc.runDir)) {
				t.Fatalf("path %q does not stay under runDir %q", got, tc.runDir)
			}
		})
	}
}
