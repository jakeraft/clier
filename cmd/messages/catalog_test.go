package messages

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

// TestEveryKindHasTemplate guarantees the catalog stays in sync with
// the domain.Kind enum. Adding a Kind without a template is a build-
// equivalent failure.
func TestEveryKindHasTemplate(t *testing.T) {
	for _, k := range domain.AllKinds() {
		if !HasTemplate(k) {
			t.Errorf("messages catalog missing template for Kind %q", k)
		}
	}
}

// TestRenderHandlesMissingSubject ensures partially-populated Faults
// never panic and never produce an empty message.
func TestRenderHandlesMissingSubject(t *testing.T) {
	for _, k := range domain.AllKinds() {
		f := &domain.Fault{Kind: k}
		r := Render(f)
		if r.Message == "" {
			t.Errorf("Render(%s) produced empty message", k)
		}
	}
}
