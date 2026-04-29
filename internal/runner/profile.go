package runner

// agentProfile carries the per-agent-type tmux hints the CLI keeps after
// ADR-0002 §6 amendment moved protocol injection to the server. Only
// readiness detection and graceful exit live here now; everything else
// (paths, settings, instruction file, protocol args) is server-owned.
type agentProfile struct {
	// readyMarker is a substring of the tmux pane title that appears once
	// the vendor TUI has finished bootstrapping. Empty disables polling
	// (the runner just trusts the launch).
	readyMarker string

	// exitCommand is the message sent to the agent before kill-session so
	// it flushes any in-flight state. Empty skips the graceful step.
	exitCommand string
}

func profileFor(agentType string) agentProfile {
	switch agentType {
	case "claude":
		return agentProfile{readyMarker: "Claude", exitCommand: "/exit"}
	case "codex":
		return agentProfile{readyMarker: "", exitCommand: "/exit"}
	}
	return agentProfile{}
}
