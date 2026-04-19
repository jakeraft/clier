package app

import (
	"errors"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/domain"
)

func TestTranslatePassesFaultsThrough(t *testing.T) {
	in := &domain.Fault{Kind: domain.KindRunNotFound}
	out := Translate(in)
	var f *domain.Fault
	if !errors.As(out, &f) || f.Kind != domain.KindRunNotFound {
		t.Fatalf("expected pass-through, got %v", out)
	}
}

func TestTranslateAPIError(t *testing.T) {
	apiErr := &api.Error{
		StatusCode: 404,
		Status: &api.Status{
			Code:   404,
			Reason: api.ReasonResourceNotFound,
			Details: &api.StatusDetails{
				Owner: "jake",
				Name:  "missing",
			},
		},
	}
	out := Translate(apiErr)
	var f *domain.Fault
	if !errors.As(out, &f) {
		t.Fatalf("expected Fault, got %T", out)
	}
	if f.Kind != domain.KindResourceNotFound {
		t.Errorf("kind = %q, want %q", f.Kind, domain.KindResourceNotFound)
	}
	if f.Subject["owner"] != "jake" || f.Subject["name"] != "missing" {
		t.Errorf("subject = %v", f.Subject)
	}
}

func TestTranslateTerminalSentinels(t *testing.T) {
	cases := []struct {
		err  error
		want domain.Kind
	}{
		{&terminal.ErrNoTTY{}, domain.KindNotATerminal},
		{&terminal.ErrSessionGone{Session: "sess-1"}, domain.KindRunInactive},
	}
	for _, c := range cases {
		out := Translate(c.err)
		var f *domain.Fault
		if !errors.As(out, &f) || f.Kind != c.want {
			t.Errorf("Translate(%T) = %v, want kind %q", c.err, out, c.want)
		}
	}
}

func TestTranslateConnRefused(t *testing.T) {
	err := &api.ConnRefusedError{Cause: errors.New("dial tcp 127.0.0.1:8080: connect: connection refused")}
	out := Translate(err)
	var f *domain.Fault
	if !errors.As(out, &f) || f.Kind != domain.KindServerUnreachable {
		t.Errorf("Translate(ConnRefusedError) = %v, want %q", out, domain.KindServerUnreachable)
	}
}

func TestTranslateCobraRequiredFlag(t *testing.T) {
	cases := []struct {
		msg   string
		flags string
	}{
		{`required flag(s) "name" not set`, "name"},
		{`required flag(s) "name", "command" not set`, "name,command"},
	}
	for _, tc := range cases {
		out := Translate(errors.New(tc.msg))
		var f *domain.Fault
		if !errors.As(out, &f) || f.Kind != domain.KindInvalidArgument {
			t.Fatalf("Translate(%q) = %v, want KindInvalidArgument", tc.msg, out)
		}
		if f.Subject["flags"] != tc.flags {
			t.Errorf("subject flags = %q, want %q", f.Subject["flags"], tc.flags)
		}
	}
}

func TestTranslateUnknownErrorBecomesInternal(t *testing.T) {
	out := Translate(errors.New("something weird"))
	var f *domain.Fault
	if !errors.As(out, &f) || f.Kind != domain.KindInternal {
		t.Errorf("unknown error → %v, want KindInternal", out)
	}
}
