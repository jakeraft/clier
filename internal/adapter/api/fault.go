package api

import (
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// ConnRefusedError marks a network failure where the TCP connection to
// the server was refused (typically: server process not running).
// Carries AsFault so it flows through the single translate boundary
// like any other api-layer error.
type ConnRefusedError struct {
	Cause error
}

func (e *ConnRefusedError) Error() string {
	if e == nil || e.Cause == nil {
		return "connection refused"
	}
	return "connection refused: " + e.Cause.Error()
}

func (e *ConnRefusedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// AsFault converts a refused-connection error into its domain Fault.
func (e *ConnRefusedError) AsFault() *domain.Fault {
	return &domain.Fault{Kind: domain.KindServerUnreachable, Cause: e}
}

// AsFault converts a server-returned Error into its domain Fault.
// Bodies without a recognizable Status envelope become KindInternal so
// the presenter always has a Kind to render.
func (e *Error) AsFault() *domain.Fault {
	if e == nil {
		return nil
	}
	if e.Status == nil {
		return &domain.Fault{Kind: domain.KindInternal, Cause: e}
	}
	kind, ok := reasonToKind[e.Status.Reason]
	if !ok {
		return &domain.Fault{Kind: domain.KindInternal, Cause: e}
	}
	return &domain.Fault{Kind: kind, Subject: subjectFromStatus(e.Status), Cause: e}
}

// reasonToKind maps server-side reason enum values to domain Kinds.
// Completeness is verified by TestEveryServerReasonTranslates.
var reasonToKind = map[Reason]domain.Kind{
	ReasonUserNotFound:            domain.KindUserNotFound,
	ReasonOrgNotFound:             domain.KindOrgNotFound,
	ReasonResourceNotFound:        domain.KindResourceNotFound,
	ReasonResourceVersionNotFound: domain.KindResourceVersionNotFound,
	ReasonOrgMemberNotFound:       domain.KindOrgMemberNotFound,
	ReasonTokenNotFound:           domain.KindTokenNotFound,
	ReasonResourceNameTaken:       domain.KindResourceNameTaken,
	ReasonOrgMemberExists:         domain.KindOrgMemberExists,
	ReasonInvalidArgument:         domain.KindInvalidArgument,
	ReasonAuthRequired:            domain.KindAuthRequired,
	ReasonAuthFailed:              domain.KindAuthFailed,
	ReasonInvalidOAuthState:       domain.KindInvalidOAuthState,
	ReasonForbidden:               domain.KindForbidden,
	ReasonNotOrgMember:            domain.KindNotOrgMember,
	ReasonNotOrgOwner:             domain.KindNotOrgOwner,
	ReasonNotTeamResource:         domain.KindNotTeamResource,
	ReasonInternal:                domain.KindInternal,
}

// ReasonToKind exposes the translation table for tests that verify
// every server reason has a domain mapping.
func ReasonToKind() map[Reason]domain.Kind { return reasonToKind }

func subjectFromStatus(s *Status) map[string]string {
	if s == nil {
		return nil
	}
	out := map[string]string{}
	if s.Details != nil {
		if s.Details.Owner != "" {
			out["owner"] = s.Details.Owner
		}
		if s.Details.Name != "" {
			out["name"] = s.Details.Name
		}
		if s.Details.Kind != "" {
			out["resource_kind"] = s.Details.Kind
		}
		if s.Details.Version > 0 {
			out["version"] = strconv.Itoa(s.Details.Version)
		}
		if len(s.Details.Causes) > 0 {
			out["detail"] = joinCauses(s.Details.Causes)
		}
	}
	if _, ok := out["detail"]; !ok && s.Message != "" {
		out["detail"] = s.Message
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func joinCauses(causes []StatusCause) string {
	parts := make([]string, 0, len(causes))
	for _, c := range causes {
		switch {
		case c.Field != "" && c.Message != "":
			parts = append(parts, c.Field+": "+c.Message)
		case c.Message != "":
			parts = append(parts, c.Message)
		}
	}
	return strings.Join(parts, "; ")
}
