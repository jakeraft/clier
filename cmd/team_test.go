package cmd

import (
	"reflect"
	"testing"

	"github.com/jakeraft/clier/internal/api"
)

// strPtr returns a pointer to v — convenience for the per-field flag args
// in buildTeamPatch tests, which need to distinguish nil ("flag absent")
// from a pointer to an empty string ("flag set to ”").
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
			if got.raw != nil {
				t.Fatalf("raw path should be nil for per-field, got %q", got.raw)
			}
			if !reflect.DeepEqual(got.fields, tc.want) {
				t.Errorf("got %+v want %+v", got.fields, tc.want)
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
		if _, present := got.fields["subteams"]; present {
			t.Errorf("subteams should be absent when --subteam not passed: %+v", got.fields)
		}
	})
	t.Run("flag passed empty clears", func(t *testing.T) {
		got, err := buildTeamPatch(nil, nil, nil, nil, nil, true, "")
		if err != nil {
			t.Fatal(err)
		}
		subs, ok := got.fields["subteams"].([]api.TeamKey)
		if !ok {
			t.Fatalf("subteams should be []api.TeamKey, got %T", got.fields["subteams"])
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
		subs := got.fields["subteams"].([]api.TeamKey)
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

func TestBuildTeamPatch_patchJsonRawPassthrough(t *testing.T) {
	// --patch-json is the escape hatch — bytes are forwarded untouched
	// to the server's parser. CLI-side unmarshal would be a duplicate
	// validation that drifts from the server's spec; shape errors land
	// as 400 Malformed request from the server with a precise json
	// offset.
	literal := `{"description":"from-json","subteams":[{"namespace":"x","name":"y"}]}`
	got, err := buildTeamPatch(nil, nil, nil, nil, nil, false, literal)
	if err != nil {
		t.Fatalf("buildTeamPatch: %v", err)
	}
	if got.fields != nil {
		t.Errorf("fields should be nil when --patch-json set, got %+v", got.fields)
	}
	if string(got.raw) != literal {
		t.Errorf("raw mismatch:\ngot  %q\nwant %q", got.raw, literal)
	}
}

func TestBuildTeamPatch_patchJsonAndFlagsMutex(t *testing.T) {
	// Mixing --patch-json with per-field flags used to silently
	// discard the per-field values; now it's rejected loudly.
	if _, err := buildTeamPatch(strPtr("from-flag"), nil, nil, nil, nil, false,
		`{"description":"from-json"}`); err == nil {
		t.Fatal("expected mutex error when --patch-json + per-field flag both present")
	}
}
