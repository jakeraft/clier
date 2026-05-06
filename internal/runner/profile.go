package runner

// agentProfile carries the per-agent-type tmux hints the CLI owns —
// the readiness marker for the launch handshake and the trust-prompt
// response for vendors that gate first-launch on a directory-trust
// dialog. Server owns protocol injection (ADR-0002 §6); the CLI only
// keeps the per-vendor surface hints that depend on the local TUI.
type agentProfile struct {
	// readyMarker is a substring of the tmux pane title that appears once
	// the vendor TUI has finished bootstrapping. Empty disables polling
	// (the runner just trusts the launch).
	readyMarker string

	// exitCommand is the message sent to the agent before kill-session so
	// it flushes any in-flight state. Empty skips the graceful step.
	exitCommand string

	// trustResponse is the keystroke(s) the runner sends to the agent's
	// pane immediately after launch to dismiss a vendor trust prompt.
	// Empty disables the auto-response. Codex blocks on "Do you trust
	// this directory?" with "1. Yes, continue", so the runner sends "1"
	// + Enter — the run dir is fresh every launch and codex has no
	// native flag to skip this prompt (ADR-0002 §8).
	trustResponse string
}

func profileFor(agentType string) agentProfile {
	switch agentType {
	case "claude":
		return agentProfile{readyMarker: "Claude", exitCommand: "/exit"}
	case "codex":
		return agentProfile{
			readyMarker:   "",
			exitCommand:   "/exit",
			trustResponse: "1",
		}
	}
	return agentProfile{}
}
