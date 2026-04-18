package cmd

import (
	"errors"
	"testing"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

func TestParseNamespaceAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    remoteapi.NamespaceAccess
		wantErr domain.Kind
	}{
		{name: "default empty is public", input: "", want: remoteapi.NamespaceAccessPublic},
		{name: "public", input: "public", want: remoteapi.NamespaceAccessPublic},
		{name: "private", input: "private", want: remoteapi.NamespaceAccessPrivate},
		{name: "trim and case fold", input: " Private ", want: remoteapi.NamespaceAccessPrivate},
		{name: "invalid", input: "members-only", wantErr: domain.KindInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseNamespaceAccess(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				var fault *domain.Fault
				if !errors.As(err, &fault) || fault.Kind != tt.wantErr {
					t.Fatalf("error = %v, want kind %s", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
