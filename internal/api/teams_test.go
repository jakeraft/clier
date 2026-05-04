package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// recorder captures the request the Client made so each test can assert
// method / path / query / body in isolation. Tests build one ad-hoc with
// the response body they expect to decode.
type recorder struct {
	method   string
	path     string
	rawQuery string
	body     []byte
	auth     string
}

func newServer(t *testing.T, status int, responseBody string) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.Path
		rec.rawQuery = r.URL.RawQuery
		rec.auth = r.Header.Get("Authorization")
		rec.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		if responseBody != "" {
			_, _ = io.WriteString(w, responseBody)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func TestClientListTeams_composesQueryAndDecodes(t *testing.T) {
	const body = `{
		"data": [
			{
				"namespace": "alice",
				"name": "frontend",
				"description": "react app",
				"agent_type": "claude",
				"command": "claude",
				"git_repo_url": "https://github.com/alice/frontend",
				"git_subpath": "",
				"protocol": "# Team Protocol",
				"subteams": [],
				"created_at": "2026-04-30T10:00:00Z",
				"updated_at": "2026-04-30T10:00:00Z",
				"layout": {"instruction_path":"CLAUDE.md","skills_dir_path":".claude/skills","settings_path":".claude/settings.json"},
				"namespace_profile": {"name":"alice","avatar_url":"https://github.com/alice.png"},
				"star": {"starred": true, "count": 7}
			}
		],
		"meta": {"has_next": true, "next_cursor": "next-token"}
	}`
	srv, rec := newServer(t, http.StatusOK, body)

	c := New(srv.URL, "tok")
	got, err := c.ListTeams(ListTeamsQuery{
		Namespace: "alice",
		AgentType: "claude",
		Sort:      "stars_desc",
		Q:         "front end",
		PageSize:  20,
		PageToken: "cursor-1",
	})
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if rec.method != http.MethodGet || rec.path != "/api/v1/teams" {
		t.Errorf("request line: %s %s", rec.method, rec.path)
	}
	if rec.auth != "Bearer tok" {
		t.Errorf("Authorization: %q", rec.auth)
	}
	wantQ := "agent_type=claude&namespace=alice&page_size=20&page_token=cursor-1&q=front+end&sort=stars_desc"
	if rec.rawQuery != wantQ {
		t.Errorf("query: got %q want %q", rec.rawQuery, wantQ)
	}
	if len(got.Data) != 1 || got.Data[0].Name != "frontend" || got.Data[0].Star.Count != 7 {
		t.Errorf("decoded payload mismatch: %+v", got.Data)
	}
	if !got.Meta.HasNext || got.Meta.NextCursor != "next-token" {
		t.Errorf("meta: %+v", got.Meta)
	}
}

func TestClientListTeams_emptyQueryHasNoQuestionMark(t *testing.T) {
	srv, rec := newServer(t, http.StatusOK, `{"data":[],"meta":{"has_next":false,"next_cursor":""}}`)

	if _, err := New(srv.URL, "").ListTeams(ListTeamsQuery{}); err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if rec.rawQuery != "" {
		t.Errorf("expected no query params, got %q", rec.rawQuery)
	}
	if rec.auth != "" {
		t.Errorf("public list should send no Authorization, got %q", rec.auth)
	}
}

func TestClientGetTeam_pathAndDecode(t *testing.T) {
	srv, rec := newServer(t, http.StatusOK, `{"namespace":"alice","name":"frontend","star":{"starred":false,"count":0}}`)

	got, err := New(srv.URL, "").GetTeam("alice", "frontend")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if rec.method != http.MethodGet || rec.path != "/api/v1/teams/alice/frontend" {
		t.Errorf("request line: %s %s", rec.method, rec.path)
	}
	if got.Namespace != "alice" || got.Name != "frontend" {
		t.Errorf("decoded: %+v", got)
	}
}

func TestClientCreateTeam_postsBody(t *testing.T) {
	srv, rec := newServer(t, http.StatusCreated, `{"namespace":"alice","name":"frontend"}`)

	req := CreateTeamRequest{
		Namespace:  "alice",
		Name:       "frontend",
		AgentType:  "claude",
		Command:    "claude --resume",
		GitRepoURL: "https://github.com/alice/frontend",
		Subteams:   []TeamKey{{Namespace: "alice", Name: "shared"}},
	}
	if _, err := New(srv.URL, "tok").CreateTeam(req); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if rec.method != http.MethodPost || rec.path != "/api/v1/teams" {
		t.Errorf("request line: %s %s", rec.method, rec.path)
	}
	var sent CreateTeamRequest
	if err := json.Unmarshal(rec.body, &sent); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if sent.Namespace != "alice" || sent.Name != "frontend" || len(sent.Subteams) != 1 {
		t.Errorf("body fields: %+v", sent)
	}
}

func TestClientUpdateTeam_patchesMergeBody(t *testing.T) {
	srv, rec := newServer(t, http.StatusOK, `{"namespace":"alice","name":"frontend","description":"new"}`)

	patch := map[string]any{"description": "new", "git_subpath": ""}
	if _, err := New(srv.URL, "tok").UpdateTeam("alice", "frontend", patch); err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	if rec.method != http.MethodPatch || rec.path != "/api/v1/teams/alice/frontend" {
		t.Errorf("request line: %s %s", rec.method, rec.path)
	}
	var sent map[string]any
	if err := json.Unmarshal(rec.body, &sent); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if sent["description"] != "new" {
		t.Errorf("description not patched: %+v", sent)
	}
	if _, ok := sent["git_subpath"]; !ok {
		t.Errorf("git_subpath should be preserved as empty-string set, got: %+v", sent)
	}
}

func TestClientDeleteTeam_noContent(t *testing.T) {
	srv, rec := newServer(t, http.StatusNoContent, "")

	if err := New(srv.URL, "tok").DeleteTeam("alice", "frontend"); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	if rec.method != http.MethodDelete || rec.path != "/api/v1/teams/alice/frontend" {
		t.Errorf("request line: %s %s", rec.method, rec.path)
	}
}

func TestClientStarTeam_putsToStarPath(t *testing.T) {
	srv, rec := newServer(t, http.StatusNoContent, "")

	if err := New(srv.URL, "tok").StarTeam("alice", "frontend"); err != nil {
		t.Fatalf("StarTeam: %v", err)
	}
	if rec.method != http.MethodPut || rec.path != "/api/v1/teams/alice/frontend/star" {
		t.Errorf("request line: %s %s", rec.method, rec.path)
	}
}

func TestClientUnstarTeam_deletesStarPath(t *testing.T) {
	srv, rec := newServer(t, http.StatusNoContent, "")

	if err := New(srv.URL, "tok").UnstarTeam("alice", "frontend"); err != nil {
		t.Fatalf("UnstarTeam: %v", err)
	}
	if rec.method != http.MethodDelete || rec.path != "/api/v1/teams/alice/frontend/star" {
		t.Errorf("request line: %s %s", rec.method, rec.path)
	}
}
