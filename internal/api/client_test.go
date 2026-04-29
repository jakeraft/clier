package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientResolveTeam_decodesRunManifest(t *testing.T) {
	const body = `{
		"mounts": [
			{"name":"jakeraft.clier-qa-claude","git_repo_url":"https://github.com/jakeraft/clier-qa","git_subpath":"teams/clier-qa-claude"}
		],
		"agents": [
			{"id":"jakeraft.clier-qa-claude","window":0,"mount":"jakeraft.clier-qa-claude","cwd":"jakeraft.clier-qa-claude/teams/clier-qa-claude","command":"CLIER_AGENT= claude","args":["--append-system-prompt","# Team Protocol\n"],"agent_type":"claude"}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/teams/jakeraft/clier-qa-claude/resolve" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("public endpoint should send no Authorization header, got %q", r.Header.Get("Authorization"))
		}
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	got, err := c.ResolveTeam("jakeraft", "clier-qa-claude")
	if err != nil {
		t.Fatalf("ResolveTeam: %v", err)
	}
	if len(got.Mounts) != 1 || got.Mounts[0].Name != "jakeraft.clier-qa-claude" {
		t.Errorf("mounts: %+v", got.Mounts)
	}
	if len(got.Agents) != 1 {
		t.Fatalf("agents: %+v", got.Agents)
	}
	a := got.Agents[0]
	if a.AgentType != "claude" || len(a.Args) != 2 || a.Args[0] != "--append-system-prompt" {
		t.Errorf("agent: %+v", a)
	}
	if !strings.Contains(a.Args[1], "# Team Protocol") {
		t.Errorf("args[1] should carry protocol markdown: %q", a.Args[1])
	}
}

func TestClientResolveTeam_authBearerInjected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-xyz" {
			t.Errorf("expected Bearer token-xyz, got %q", r.Header.Get("Authorization"))
		}
		_, _ = io.WriteString(w, `{"mounts":[],"agents":[]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "token-xyz")
	if _, err := c.ResolveTeam("ns", "team"); err != nil {
		t.Fatalf("ResolveTeam: %v", err)
	}
}

func TestClientErrorParsesProblemCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPreconditionFailed)
		_, _ = io.WriteString(w, `{"code":"FAILED_PRECONDITION","detail":"login not yet confirmed by user","status":412}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.AuthDeviceComplete("dev-code")
	if err == nil {
		t.Fatal("want error")
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *api.Error, got %T", err)
	}
	if apiErr.StatusCode != 412 {
		t.Errorf("status: got %d, want 412", apiErr.StatusCode)
	}
	if apiErr.Code() != "FAILED_PRECONDITION" {
		t.Errorf("code: got %q, want FAILED_PRECONDITION", apiErr.Code())
	}
}

func TestClientErrorOnNonJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "<html>upstream broken</html>")
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.AuthMe()
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *api.Error, got %T (%v)", err, err)
	}
	if apiErr.Code() != "" {
		t.Errorf("non-JSON body should yield empty Code(), got %q", apiErr.Code())
	}
	if !strings.Contains(apiErr.Body, "upstream broken") {
		t.Errorf("Body should preserve the raw payload, got %q", apiErr.Body)
	}
}

func TestClient204LogoutIsNoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auth/logout" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "session-token")
	if err := c.AuthLogout(); err != nil {
		t.Fatalf("AuthLogout: %v", err)
	}
}

func TestClientAuthDeviceStartRoundtrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/device/start" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(DeviceAuthorization{
			DeviceCode:      "dc",
			UserCode:        "WXYZ-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	got, err := c.AuthDeviceStart()
	if err != nil {
		t.Fatalf("AuthDeviceStart: %v", err)
	}
	if got.UserCode != "WXYZ-1234" || got.Interval != 5 {
		t.Errorf("unexpected response: %+v", got)
	}
}
