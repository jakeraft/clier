package terminal

import "github.com/jakeraft/clier/internal/domain"

// AsFault converts the sentinel into its domain Fault.
func (*ErrNoTTY) AsFault() *domain.Fault {
	return &domain.Fault{Kind: domain.KindNotATerminal}
}

// AsFault converts the sentinel into its domain Fault, carrying the
// session identifier as subject context when available.
func (e *ErrSessionGone) AsFault() *domain.Fault {
	var subj map[string]string
	if e != nil && e.Session != "" {
		subj = map[string]string{"session": e.Session}
	}
	return &domain.Fault{Kind: domain.KindRunInactive, Subject: subj, Cause: e}
}
