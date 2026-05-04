package runner

// agentProfile carries the per-agent-type tmux hints the CLI keeps after
// ADR-0002 §6 amendment moved protocol injection to the server. Now also
// holds the trust-prompt response for vendors that gate first-launch on
// a directory-trust dialog (codex 0.121+ — issue #19426 still open).
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
	// Empty disables the auto-response (claude has no trust gate at
	// launch). Codex 0.121+ blocks on "Do you trust this directory?" with
	// "1. Yes, continue" so the runner sends "1" + Enter — codex has no
	// native flag to skip this prompt and the run dir is fresh every
	// launch (ADR-0002 §8 amendment).
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
