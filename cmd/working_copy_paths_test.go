package cmd

import "testing"

func TestValidateOwner_RejectsDotPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		owner   string
		wantErr bool
	}{
		{"jakeraft", false},
		{"@clier", false},
		{".runs", true},
		{".hidden", true},
	}
	for _, tc := range cases {
		err := validateOwner(tc.owner)
		if tc.wantErr && err == nil {
			t.Errorf("validateOwner(%q) = nil, want error", tc.owner)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("validateOwner(%q) = %v, want nil", tc.owner, err)
		}
	}
}
