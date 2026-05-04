package cmd

import (
	"reflect"
	"testing"

	"github.com/jakeraft/clier/internal/api"
)

// strPtr returns a pointer to v — convenience for the per-field flag args
// in buildTeamPatch tests, which need to distinguish nil ("flag absent")
// from a pointer to an empty string ("flag set to ''").
func strPtr(v string) *string { return &v }

func TestBuildTeamPatch_perFieldPointers(t *testing.T) {
	cases := []struct {
		name string
		desc *string
		cmd  *string
		want map[string]any
	}{
		{
			name: "all absent",
			want: map[string]any{},
		},
		{
			name: "description set, command absent",
			desc: strPtr("new desc"),
			want: map[string]any{"description": "new desc"},
		},
		{
			name: "command set to empty string is preserved (clear)",
			cmd:  strPtr(""),
			want: map[string]any{"command": ""},
		},
		{
			name: "both set",
			desc: strPtr("d"),
			cmd:  strPtr("c"),
			want: map[string]any{"description": "d", "command": "c"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildTeamPatch(tc.desc, tc.cmd, nil, nil, nil, false, "")
			if err != nil {
				t.Fatalf("buildTeamPatch: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestBuildTeamPatch_subteamsClearVsAdd(t *testing.T) {
	t.Run("flag absent omits field", func(t *testing.T) {
		got, err := buildTeamPatch(nil, nil, nil, nil, nil, false, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, present := got["subteams"]; present {
			t.Errorf("subteams should be absent when --subteam not passed: %+v", got)
		}
	})
	t.Run("flag passed empty clears", func(t *testing.T) {
		got, err := buildTeamPatch(nil, nil, nil, nil, nil, true, "")
		if err != nil {
			t.Fatal(err)
		}
		subs, ok := got["subteams"].([]api.TeamKey)
		if !ok {
			t.Fatalf("subteams should be []api.TeamKey, got %T", got["subteams"])
		}
		if len(subs) != 0 {
			t.Errorf("expected empty slice (clear semantic), got %+v", subs)
		}
	})
	t.Run("flag passed with refs replaces", func(t *testing.T) {
		got, err := buildTeamPatch(nil, nil, nil, nil, []string{"alice/x", "bob/y"}, true, "")
		if err != nil {
			t.Fatal(err)
		}
		subs := got["subteams"].([]api.TeamKey)
		want := []api.TeamKey{{Namespace: "alice", Name: "x"}, {Namespace: "bob", Name: "y"}}
		if !reflect.DeepEqual(subs, want) {
			t.Errorf("got %+v want %+v", subs, want)
		}
	})
	t.Run("invalid ref surfaces error", func(t *testing.T) {
		_, err := buildTeamPatch(nil, nil, nil, nil, []string{"no-slash"}, true, "")
		if err == nil {
			t.Fatal("expected error for ref without slash")
		}
	})
}

func TestBuildTeamPatch_patchJsonEscapeHatchWins(t *testing.T) {
	// When --patch-json is set the per-field flags are ignored entirely so
	// operators have a single deterministic body when they need a complex
	// merge patch (e.g. nested object semantics that flag composition cannot
	// express). Verifies that the per-field "description" does NOT leak.
	got, err := buildTeamPatch(strPtr("ignored"), nil, nil, nil, nil, false,
		`{"description":"from-json","subteams":[{"namespace":"x","name":"y"}]}`)
	if err != nil {
		t.Fatalf("buildTeamPatch: %v", err)
	}
	if got["description"] != "from-json" {
		t.Errorf("--patch-json should override per-field flags, got %+v", got)
	}
	subs, ok := got["subteams"].([]any)
	if !ok || len(subs) != 1 {
		t.Errorf("subteams should be the JSON-decoded array (any), got %T %+v", got["subteams"], got["subteams"])
	}
}

func TestBuildTeamPatch_invalidPatchJson(t *testing.T) {
	if _, err := buildTeamPatch(nil, nil, nil, nil, nil, false, "{not json"); err == nil {
		t.Fatal("expected error for malformed --patch-json")
	}
}
