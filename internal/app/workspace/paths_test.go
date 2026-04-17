package workspace

import "testing"

func TestAgentWorkspaceLocalPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		owner string
		agent string
		want  string
	}{
		{name: "owner and agent", owner: "@clier", agent: "hello-codex", want: "@clier.hello-codex"},
		{name: "sanitized", owner: "team ops", agent: "review squad", want: "team-ops.review-squad"},
		{name: "missing owner", owner: "", agent: "solo", want: "solo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := AgentWorkspaceLocalPath(tc.owner, tc.agent); got != tc.want {
				t.Fatalf("AgentWorkspaceLocalPath(%q, %q) = %q, want %q", tc.owner, tc.agent, got, tc.want)
			}
		})
	}
}
