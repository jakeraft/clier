// Package present is the single CLI output sink. Success payloads are
// emitted by the command's RunE; errors flow through Emit.
//
// Both shapes are JSON envelopes so agent consumers can always parse
// stdout/stderr the same way.
package present

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/jakeraft/clier/cmd/messages"
	"github.com/jakeraft/clier/internal/domain"
)

// ErrorEnvelope is the on-the-wire shape written to stderr for any
// command failure.
type ErrorEnvelope struct {
	Error Body `json:"error"`
}

// Body carries the user-facing message plus structured context for
// agents that prefer machine handling.
type Body struct {
	Kind    string            `json:"kind"`
	Message string            `json:"message"`
	Hint    string            `json:"hint,omitempty"`
	Subject map[string]string `json:"subject,omitempty"`
}

// Emit writes the rendered error envelope to w. Any non-Fault error is
// wrapped as KindInternal so the output schema stays uniform.
//
// When the catalog-rendered message already conveys what's in
// Subject["detail"] (typical for server faults whose human message
// equals the template), the redundant key is dropped so users don't
// see the same line twice.
func Emit(w io.Writer, err error) {
	if err == nil {
		return
	}
	f := faultOf(err)
	r := messages.Render(f)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(ErrorEnvelope{Body{
		Kind:    string(f.Kind),
		Message: r.Message,
		Hint:    r.Hint,
		Subject: pruneSubject(f.Subject, r.Message),
	}})
}

// Success writes the command's success payload to stdout.
// Callers must pass a top-level JSON object payload.
func Success(w io.Writer, payload any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(payload)
}

func pruneSubject(subject map[string]string, message string) map[string]string {
	if len(subject) == 0 {
		return nil
	}
	out := make(map[string]string, len(subject))
	for k, v := range subject {
		if k == "detail" && v == message {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func faultOf(err error) *domain.Fault {
	var f *domain.Fault
	if errors.As(err, &f) {
		return f
	}
	return &domain.Fault{Kind: domain.KindInternal, Cause: err}
}
