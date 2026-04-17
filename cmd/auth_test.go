package cmd

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/auth"
)

func TestAuthStatusResult_NotLoggedIn(t *testing.T) {
	msg, ok := authStatusResult(nil, nil, nil)
	if ok {
		t.Fatal("ok = true for missing credentials, want false")
	}
	if msg != "Not logged in." {
		t.Fatalf("msg = %q", msg)
	}
}

func TestAuthStatusResult_LoggedIn(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	user := &api.UserResponse{Name: "jakeraft"}
	msg, ok := authStatusResult(creds, user, nil)
	if !ok {
		t.Fatal("ok = false for server-confirmed login, want true")
	}
	if msg != "Logged in as jakeraft" {
		t.Fatalf("msg = %q", msg)
	}
}

func TestAuthStatusResult_TokenInvalid(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		err := &api.Error{StatusCode: code, Body: "unauthorized"}
		msg, ok := authStatusResult(creds, nil, err)
		if ok {
			t.Errorf("status %d: ok = true, want false for invalid token", code)
		}
		if !strings.Contains(msg, "invalid or expired") {
			t.Errorf("status %d: msg = %q, want explicit invalid/expired wording", code, msg)
		}
		if strings.Contains(msg, "may be expired") {
			t.Errorf("status %d: msg = %q, should not contain ambiguous wording", code, msg)
		}
		if !strings.Contains(msg, "clier auth login") {
			t.Errorf("status %d: msg = %q, want re-auth instruction", code, msg)
		}
	}
}

func TestAuthStatusResult_ServerUnreachable(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	netErr := errors.New("do: dial tcp: connection refused")
	msg, ok := authStatusResult(creds, nil, netErr)
	if ok {
		t.Fatal("ok = true for network error, want false")
	}
	if !strings.Contains(msg, "Unable to verify login for jakeraft") {
		t.Errorf("msg = %q, want unreachable message including login", msg)
	}
	if !strings.Contains(msg, "connection refused") {
		t.Errorf("msg = %q, want to surface underlying error", msg)
	}
}

func TestAuthStatusResult_ServerError5xx(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	apiErr := &api.Error{StatusCode: http.StatusInternalServerError, Body: "boom"}
	msg, ok := authStatusResult(creds, nil, apiErr)
	if ok {
		t.Fatal("ok = true for 5xx, want false")
	}
	if !strings.Contains(msg, "Unable to verify login") {
		t.Errorf("msg = %q, want unreachable wording for 5xx", msg)
	}
	if strings.Contains(msg, "invalid or expired") {
		t.Errorf("msg = %q, 5xx should not be reported as invalid token", msg)
	}
}
