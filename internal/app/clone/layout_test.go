package clone

import "testing"

func TestResolveRepoDirName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		repoURL  string
		fallback string
		want     string
	}{
		{name: "https", repoURL: "https://github.com/jakeraft/clier_todo.git", fallback: "reviewer", want: "clier_todo"},
		{name: "ssh", repoURL: "git@github.com:jakeraft/clier_todo.git", fallback: "reviewer", want: "clier_todo"},
		{name: "fallback", repoURL: "", fallback: "reviewer", want: "reviewer"},
		{name: "sanitized fallback", repoURL: "", fallback: "review squad", want: "review-squad"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveRepoDirName(tc.repoURL, tc.fallback); got != tc.want {
				t.Fatalf("ResolveRepoDirName(%q, %q) = %q, want %q", tc.repoURL, tc.fallback, got, tc.want)
			}
		})
	}
}
