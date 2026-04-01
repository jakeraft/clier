package domain

import (
	"strings"
	"testing"
)

func TestDefaultProtocol(t *testing.T) {
	for _, want := range []string{
		"# Team Protocol",
		"clier sprint context",
		"clier message send",
		"leader",
		"worker",
		"peer",
	} {
		if !strings.Contains(DefaultProtocol, want) {
			t.Errorf("DefaultProtocol missing %q", want)
		}
	}
}
