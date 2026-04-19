// Package app contains use-case orchestration that sits between the
// CLI and the adapters. Translate is the single error boundary that
// normalizes adapter and library errors into domain.Fault values
// consumable by the message catalog.
package app

import (
	"errors"
	"regexp"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// faultable is the contract an adapter error implements to provide its
// own domain Fault mapping. Keeping the mapping next to the error type
// means this package doesn't need to know every adapter's error shape.
type faultable interface {
	AsFault() *domain.Fault
}

// External library errors (cobra) don't implement faultable, so we
// match their stable message shapes here. Add new patterns only when
// the upstream library gives us no semantic alternative.
var (
	cobraUnknownCommandRE     = regexp.MustCompile(`^unknown command "([^"]*)"`)
	cobraRequiredFlagPrefixRE = regexp.MustCompile(`^required flag\(s\) `)
	cobraQuotedNameRE         = regexp.MustCompile(`"([^"]+)"`)
)

// Translate normalizes any error reaching the presenter into a
// domain.Fault. Adapter errors self-convert via AsFault; library
// errors fall through to pattern matching; anything else becomes
// KindInternal so the presenter always has a Kind to render.
func Translate(err error) error {
	if err == nil {
		return nil
	}

	var fault *domain.Fault
	if errors.As(err, &fault) {
		return err
	}

	var faulty faultable
	if errors.As(err, &faulty) {
		return faulty.AsFault()
	}

	msg := err.Error()
	if cobraRequiredFlagPrefixRE.MatchString(msg) {
		matches := cobraQuotedNameRE.FindAllStringSubmatch(msg, -1)
		flags := make([]string, 0, len(matches))
		for _, m := range matches {
			flags = append(flags, m[1])
		}
		return &domain.Fault{
			Kind:    domain.KindInvalidArgument,
			Subject: map[string]string{"flags": strings.Join(flags, ","), "detail": msg},
			Cause:   err,
		}
	}
	if m := cobraUnknownCommandRE.FindStringSubmatch(msg); m != nil {
		return &domain.Fault{
			Kind:    domain.KindUnknownCommand,
			Subject: map[string]string{"command": m[1]},
			Cause:   err,
		}
	}

	return &domain.Fault{Kind: domain.KindInternal, Cause: err}
}
