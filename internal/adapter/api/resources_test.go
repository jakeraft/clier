package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForkResourcePath(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"kind":"team","metadata":{"name":"hello","owner_name":"jake","updated_at":"2026-04-19T00:00:00Z"},"spec":{}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	if _, err := client.ForkResource(KindTeam, "@clier", "hello-claude"); err != nil {
		t.Fatalf("ForkResource: %v", err)
	}

	want := "/api/v1/orgs/@clier/teams/hello-claude/fork"
	if gotPath != want {
		t.Fatalf("path = %q, want %q", gotPath, want)
	}
}
