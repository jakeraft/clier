package api

import "testing"

// TestEveryServerReasonTranslates verifies every server Reason enum
// value has a domain.Kind mapping. Pairs with AllReasons(), which must
// stay in sync with the OpenAPI enum.
func TestEveryServerReasonTranslates(t *testing.T) {
	tbl := ReasonToKind()
	for _, r := range AllReasons() {
		if _, ok := tbl[r]; !ok {
			t.Errorf("missing mapping for server reason %q", r)
		}
	}
}
