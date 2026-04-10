package api

import (
	"encoding/json"
	"testing"
)

func TestClaudeMdResponse_SummaryField(t *testing.T) {
	t.Parallel()

	raw := `{"id":1,"owner_id":1,"name":"test","content":"c","summary":"short desc","visibility":0,"is_fork":false,"fork_count":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","owner_login":"me"}`
	var resp ClaudeMdResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Summary != "short desc" {
		t.Fatalf("Summary = %q, want %q", resp.Summary, "short desc")
	}
}

func TestClaudeMdWriteRequest_SummaryField(t *testing.T) {
	t.Parallel()

	req := ClaudeMdWriteRequest{Name: "n", Content: "c", Summary: "s"}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["summary"] != "s" {
		t.Fatalf("summary = %v, want %q", m["summary"], "s")
	}
}

func TestClaudeMdWriteRequest_SummaryOmittedWhenEmpty(t *testing.T) {
	t.Parallel()

	req := ClaudeMdWriteRequest{Name: "n", Content: "c"}
	b, _ := json.Marshal(req)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["summary"]; ok {
		t.Fatal("expected summary to be omitted when empty")
	}
}

func TestClaudeMdPatchRequest_OmitsUnsetFields(t *testing.T) {
	t.Parallel()

	summary := "new summary"
	req := ClaudeMdPatchRequest{Summary: &summary}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["name"]; ok {
		t.Fatal("expected name to be omitted")
	}
	if _, ok := m["content"]; ok {
		t.Fatal("expected content to be omitted")
	}
	if m["summary"] != "new summary" {
		t.Fatalf("summary = %v, want %q", m["summary"], "new summary")
	}
}
