// Package app contains use-case orchestration that sits between the
// CLI and the adapters. The Translate function defined here is the
// boundary that converts adapter-specific Failures into domain.Fault
// values consumable by the message catalog.
package app

import (
	"errors"
	"regexp"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/domain"
)

// cobra emits errors of the form: unknown command "foo" for "clier"
// We surface these as KindUnknownCommand so users get a clear hint
// instead of the catch-all "internal error" message.
var cobraUnknownCommandRE = regexp.MustCompile(`^unknown command "([^"]*)"`)

// reasonToKind maps server-side reason enum values to domain Kinds.
// Verified for completeness against the OpenAPI Reason enum by
// TestEveryServerReasonTranslates.
var reasonToKind = map[remoteapi.Reason]domain.Kind{
	remoteapi.ReasonUserNotFound:            domain.KindUserNotFound,
	remoteapi.ReasonOrgNotFound:             domain.KindOrgNotFound,
	remoteapi.ReasonResourceNotFound:        domain.KindResourceNotFound,
	remoteapi.ReasonResourceVersionNotFound: domain.KindResourceVersionNotFound,
	remoteapi.ReasonOrgMemberNotFound:       domain.KindOrgMemberNotFound,
	remoteapi.ReasonTokenNotFound:           domain.KindTokenNotFound,
	remoteapi.ReasonResourceNameTaken:       domain.KindResourceNameTaken,
	remoteapi.ReasonOrgMemberExists:         domain.KindOrgMemberExists,
	remoteapi.ReasonInvalidArgument:         domain.KindInvalidArgument,
	remoteapi.ReasonAuthRequired:            domain.KindAuthRequired,
	remoteapi.ReasonAuthFailed:              domain.KindAuthFailed,
	remoteapi.ReasonInvalidOAuthState:       domain.KindInvalidOAuthState,
	remoteapi.ReasonForbidden:               domain.KindForbidden,
	remoteapi.ReasonNotOrgMember:            domain.KindNotOrgMember,
	remoteapi.ReasonNotOrgOwner:             domain.KindNotOrgOwner,
	remoteapi.ReasonNotTeamResource:         domain.KindNotTeamResource,
	remoteapi.ReasonInternal:                domain.KindInternal,
}

// ReasonToKind exposes the translation table for tests that verify
// every server reason has a domain mapping.
func ReasonToKind() map[remoteapi.Reason]domain.Kind { return reasonToKind }

// Translate converts adapter-level errors into domain.Fault values.
// Faults already produced by upstream code pass through unchanged.
// Unknown errors are wrapped as KindInternal so the presenter never
// receives a bare error.
func Translate(err error) error {
	if err == nil {
		return nil
	}

	// Already domain-typed — leave alone.
	var fault *domain.Fault
	if errors.As(err, &fault) {
		return err
	}

	if f := translateAPI(err); f != nil {
		return f
	}
	if f := translateTerminal(err); f != nil {
		return f
	}
	if remoteapi.IsConnRefused(err) {
		return &domain.Fault{Kind: domain.KindServerUnreachable, Cause: err}
	}

	if m := cobraUnknownCommandRE.FindStringSubmatch(err.Error()); m != nil {
		return &domain.Fault{
			Kind:    domain.KindUnknownCommand,
			Subject: map[string]string{"command": m[1]},
			Cause:   err,
		}
	}

	return &domain.Fault{Kind: domain.KindInternal, Cause: err}
}

func translateAPI(err error) *domain.Fault {
	var apiErr *remoteapi.Error
	if !errors.As(err, &apiErr) {
		return nil
	}
	if apiErr.Status == nil {
		return &domain.Fault{Kind: domain.KindInternal, Cause: err}
	}
	kind, ok := reasonToKind[apiErr.Status.Reason]
	if !ok {
		return &domain.Fault{Kind: domain.KindInternal, Cause: err}
	}
	subj := subjectFromStatus(apiErr.Status)
	return &domain.Fault{Kind: kind, Subject: subj, Cause: err}
}

func translateTerminal(err error) *domain.Fault {
	var noTTY *terminal.ErrNoTTY
	if errors.As(err, &noTTY) {
		return &domain.Fault{Kind: domain.KindNotATerminal, Cause: err}
	}
	var sessionGone *terminal.ErrSessionGone
	if errors.As(err, &sessionGone) {
		subj := map[string]string{}
		if sessionGone.Session != "" {
			subj["session"] = sessionGone.Session
		}
		return &domain.Fault{Kind: domain.KindRunInactive, Subject: subj, Cause: err}
	}
	return nil
}

func subjectFromStatus(s *remoteapi.Status) map[string]string {
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
			out["version"] = itoa(s.Details.Version)
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
