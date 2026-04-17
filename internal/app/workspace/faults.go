package workspace

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// internalFault wraps a corruption-like detail (workspace state
// inconsistency, missing local entries, etc.) into a KindInternal
// Fault. The CLI presenter renders it through the catalog so users
// don't see the raw "team state X/Y not found" debug message.
func internalFault(format string, args ...any) *domain.Fault {
	return &domain.Fault{
		Kind:    domain.KindInternal,
		Subject: map[string]string{"detail": fmt.Sprintf(format, args...)},
	}
}
