package run

import domainruntime "github.com/jakeraft/clier/internal/domain/runtime"

const (
	StatusRunning = domainruntime.StatusRunning
	StatusStopped = domainruntime.StatusStopped
)

type RunPlan = domainruntime.Run
type AgentTerminal = domainruntime.AgentTerminal
type RecordedMessage = domainruntime.RecordedMessage
type RecordedNote = domainruntime.RecordedNote

func NewPlan(runID, sessionName, workingCopyPath string, plans []AgentTerminal) *RunPlan {
	return domainruntime.NewRun(runID, sessionName, workingCopyPath, plans)
}
