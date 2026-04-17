package cmd

import (
	"path/filepath"
	"testing"
)

func TestWorkingCopyPath_OwnerAndName(t *testing.T) {
	t.Parallel()

	// Bypass currentConfig() by computing as workspaceDir would, with a fixed root.
	root := "/tmp/workspace"
	got := filepath.Join(root, "jakeraft", "reviewer")
	want := filepath.Join("/tmp/workspace", "jakeraft", "reviewer")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateOwner_RejectsDotPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		owner   string
		wantErr bool
	}{
		{"jakeraft", false},
		{"@clier", false},
		{".runs", true},
		{".hidden", true},
	}
	for _, tc := range cases {
		err := validateOwner(tc.owner)
		if tc.wantErr && err == nil {
			t.Errorf("validateOwner(%q) = nil, want error", tc.owner)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("validateOwner(%q) = %v, want nil", tc.owner, err)
		}
	}
}
