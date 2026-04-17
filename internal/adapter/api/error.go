package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
)

// Reason is the machine-readable error code returned by the server.
// Mirrors the OpenAPI Status.reason enum.
type Reason string

const (
	ReasonUserNotFound            Reason = "USER_NOT_FOUND"
	ReasonOrgNotFound             Reason = "ORG_NOT_FOUND"
	ReasonResourceNotFound        Reason = "RESOURCE_NOT_FOUND"
	ReasonResourceVersionNotFound Reason = "RESOURCE_VERSION_NOT_FOUND"
	ReasonOrgMemberNotFound       Reason = "ORG_MEMBER_NOT_FOUND"
	ReasonTokenNotFound           Reason = "TOKEN_NOT_FOUND"
	ReasonResourceNameTaken       Reason = "RESOURCE_NAME_TAKEN"
	ReasonOrgMemberExists         Reason = "ORG_MEMBER_EXISTS"
	ReasonInvalidArgument         Reason = "INVALID_ARGUMENT"
	ReasonAuthRequired            Reason = "AUTH_REQUIRED"
	ReasonAuthFailed              Reason = "AUTH_FAILED"
	ReasonInvalidOAuthState       Reason = "INVALID_OAUTH_STATE"
	ReasonForbidden               Reason = "FORBIDDEN"
	ReasonNotOrgMember            Reason = "NOT_ORG_MEMBER"
	ReasonNotOrgOwner             Reason = "NOT_ORG_OWNER"
	ReasonNotTeamResource         Reason = "NOT_TEAM_RESOURCE"
	ReasonInternal                Reason = "INTERNAL"
)

// AllReasons returns every Reason exposed by the server's OpenAPI
// schema. Translation tests use it to verify the client maps every
// reason to a domain.Kind. Add new server reasons here in the same
// commit that adds the constant.
func AllReasons() []Reason {
	return []Reason{
		ReasonUserNotFound,
		ReasonOrgNotFound,
		ReasonResourceNotFound,
		ReasonResourceVersionNotFound,
		ReasonOrgMemberNotFound,
		ReasonTokenNotFound,
		ReasonResourceNameTaken,
		ReasonOrgMemberExists,
		ReasonInvalidArgument,
		ReasonAuthRequired,
		ReasonAuthFailed,
		ReasonInvalidOAuthState,
		ReasonForbidden,
		ReasonNotOrgMember,
		ReasonNotOrgOwner,
		ReasonNotTeamResource,
		ReasonInternal,
	}
}

// Status mirrors the server Status schema.
type Status struct {
	Code    int            `json:"code"`
	Reason  Reason         `json:"reason"`
	Message string         `json:"message"`
	Details *StatusDetails `json:"details,omitempty"`
}

// StatusDetails carries structured context about the failed resource.
type StatusDetails struct {
	Kind    string        `json:"kind,omitempty"`
	Owner   string        `json:"owner,omitempty"`
	Name    string        `json:"name,omitempty"`
	Version int           `json:"version,omitempty"`
	Causes  []StatusCause `json:"causes,omitempty"`
}

// StatusCause describes a single field-level validation cause.
type StatusCause struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// errorEnvelope mirrors the server ErrorResponse wrapper.
type errorEnvelope struct {
	Error Status `json:"error"`
}

// parseStatus extracts the structured Status from a JSON body.
// Returns nil if the body does not contain a recognizable error envelope.
func parseStatus(body []byte) *Status {
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) == 0 {
		return nil
	}
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil
	}
	if env.Error.Reason == "" && env.Error.Message == "" {
		return nil
	}
	return &env.Error
}

// IsConnRefused reports whether err originated from a refused TCP connection
// to the server (typically: server not running).
func IsConnRefused(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return true
		}
		// Fallback: some platforms wrap differently.
		if strings.Contains(opErr.Err.Error(), "connection refused") {
			return true
		}
	}
	return strings.Contains(err.Error(), "connection refused")
}

// Error returns a debug-shaped representation. End-user messages MUST
// flow through the message catalog via app.Translate + present.Emit;
// this string is only for logs and panics.
func (e *Error) Error() string {
	if e == nil {
		return "api error: <nil>"
	}
	reason := "?"
	if e.Status != nil && e.Status.Reason != "" {
		reason = string(e.Status.Reason)
	}
	return fmt.Sprintf("api error: status=%d reason=%s", e.StatusCode, reason)
}
