package cmd

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/auth"
)

func TestAuthStatusMessage_NotLoggedIn(t *testing.T) {
	got := authStatusMessage(nil, nil, nil)
	if got != "Not logged in." {
		t.Fatalf("got %q", got)
	}
}

func TestAuthStatusMessage_LoggedIn(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	user := &api.UserResponse{Name: "jakeraft"}
	got := authStatusMessage(creds, user, nil)
	if got != "Logged in as jakeraft" {
		t.Fatalf("got %q", got)
	}
}

func TestAuthStatusMessage_TokenInvalid(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		err := &api.Error{StatusCode: code, Body: "unauthorized"}
		got := authStatusMessage(creds, nil, err)
		if !strings.Contains(got, "invalid or expired") {
			t.Errorf("status %d: got %q, want message about invalid/expired token", code, got)
		}
		if strings.Contains(got, "may be expired") {
			t.Errorf("status %d: got %q, should not contain ambiguous wording", code, got)
		}
		if !strings.Contains(got, "clier auth login") {
			t.Errorf("status %d: got %q, want re-auth instruction", code, got)
		}
	}
}

func TestAuthStatusMessage_ServerUnreachable(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	netErr := errors.New("do: dial tcp: connection refused")
	got := authStatusMessage(creds, nil, netErr)
	if !strings.Contains(got, "Unable to verify login for jakeraft") {
		t.Errorf("got %q, want unreachable message including login", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("got %q, want to surface underlying error", got)
	}
}

func TestAuthStatusMessage_ServerError5xx(t *testing.T) {
	creds := &auth.Credentials{Login: "jakeraft"}
	apiErr := &api.Error{StatusCode: http.StatusInternalServerError, Body: "boom"}
	got := authStatusMessage(creds, nil, apiErr)
	if !strings.Contains(got, "Unable to verify login") {
		t.Errorf("got %q, want unreachable wording for 5xx", got)
	}
	if strings.Contains(got, "invalid or expired") {
		t.Errorf("got %q, 5xx should not be reported as invalid token", got)
	}
}
