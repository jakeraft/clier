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

func TestClientMintRun_decodesRunManifest(t *testing.T) {
	const body = `{
		"run_id": "20260430-153045-abc12345",
		"agents": [
			{
				"id": "jakeraft.hello-clier",
				"prepare": {
					"git": {
						"repo_url": "https://github.com/jakeraft/hello-clier",
						"subpath": "",
						"dest": "jakeraft.hello-clier"
					},
					"protocol": {
						"content": "# Team Protocol\n\nYou are jakeraft.hello-clier",
						"dest": "protocols/jakeraft.hello-clier.md"
					}
				},
				"run": {
					"agent_type": "claude",
					"command": "claude",
					"args": ["--append-system-prompt-file", "../protocols/jakeraft.hello-clier.md"]
				}
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/teams/jakeraft/hello-clier/runs" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("public endpoint should send no Authorization header, got %q", r.Header.Get("Authorization"))
		}
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	got, err := c.MintRun("jakeraft", "hello-clier")
	if err != nil {
		t.Fatalf("MintRun: %v", err)
	}
	if got.RunID != "20260430-153045-abc12345" {
		t.Errorf("RunID: %q", got.RunID)
	}
	if len(got.Agents) != 1 {
		t.Fatalf("agents: %+v", got.Agents)
	}
	a := got.Agents[0]
	if a.ID != "jakeraft.hello-clier" {
		t.Errorf("agent id: %q", a.ID)
	}
	if a.Prepare.Git.Dest != "jakeraft.hello-clier" {
		t.Errorf("git.dest: %q", a.Prepare.Git.Dest)
	}
	if a.Prepare.Protocol.Dest != "protocols/jakeraft.hello-clier.md" {
		t.Errorf("protocol.dest: %q", a.Prepare.Protocol.Dest)
	}
	if a.Run.AgentType != "claude" || len(a.Run.Args) != 2 || a.Run.Args[0] != "--append-system-prompt-file" {
		t.Errorf("run: %+v", a.Run)
	}
	if !strings.HasSuffix(a.Run.Args[1], "/protocols/jakeraft.hello-clier.md") {
		t.Errorf("args[1] should be a relpath to the protocol file: %q", a.Run.Args[1])
	}
	if !strings.Contains(a.Prepare.Protocol.Content, "You are jakeraft.hello-clier") {
		t.Errorf("protocol.content missing rendered body: %q", a.Prepare.Protocol.Content)
	}
}

func TestClientMintRun_authBearerInjected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-xyz" {
			t.Errorf("expected Bearer token-xyz, got %q", r.Header.Get("Authorization"))
		}
		_, _ = io.WriteString(w, `{"run_id":"x","agents":[]}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "token-xyz")
	if _, err := c.MintRun("ns", "team"); err != nil {
		t.Fatalf("MintRun: %v", err)
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

func TestClientErrorFormat_problemRendersHumanLine(t *testing.T) {
	// Server is the SSOT for the human-readable line — detail already
	// covers per-field violations (composed from errors[] server-side).
	// CLI just surfaces "<status> <title>: <detail>" with no extra
	// composition, keeping the seam between server and CLI clean.
	body := `{"type":"urn:problem:invalid-argument","title":"Invalid argument","status":422,"code":"INVALID_ARGUMENT","detail":"sort: must be one of stars_desc, stars_asc, updated_desc, updated_asc","errors":[{"field":"sort","detail":"must be one of stars_desc, stars_asc, updated_desc, updated_asc"}]}`
	e := &Error{StatusCode: 422, Body: body}

	got := e.Error()
	want := `422 Invalid argument: sort: must be one of stars_desc, stars_asc, updated_desc, updated_asc`
	if got != want {
		t.Errorf("Error():\n got: %q\nwant: %q", got, want)
	}
	if e.Code() != "INVALID_ARGUMENT" {
		t.Errorf("Code(): %q", e.Code())
	}
	// Raw envelope preserved on the public Body field for callers that
	// want to inspect errors[] / instance / type — e.g. behind --verbose.
	if e.Body != body {
		t.Errorf("Body should preserve full envelope verbatim, got %q", e.Body)
	}
}

func TestClientErrorFormat_nonProblemFallsBackToRaw(t *testing.T) {
	e := &Error{StatusCode: 500, Body: "upstream broken"}
	got := e.Error()
	want := `server returned 500: upstream broken`
	if got != want {
		t.Errorf("Error():\n got: %q\nwant: %q", got, want)
	}
}

func TestClientErrorFormat_emptyJsonBodyFallsBackToStatusText(t *testing.T) {
	// Valid JSON but no ProblemDetails fields — must not render an empty
	// summary like "500 :". Falls back to the raw body so info is kept.
	e := &Error{StatusCode: 500, Body: "{}"}
	got := e.Error()
	want := `server returned 500: {}`
	if got != want {
		t.Errorf("Error():\n got: %q\nwant: %q", got, want)
	}
}

func TestClientErrorFormat_problemWithoutTitleUsesStatusText(t *testing.T) {
	body := `{"detail":"missing session","code":"UNAUTHENTICATED"}`
	e := &Error{StatusCode: 401, Body: body}
	got := e.Error()
	want := `401 Unauthorized: missing session`
	if got != want {
		t.Errorf("Error():\n got: %q\nwant: %q", got, want)
	}
}
