package workspace

import "fmt"

const (
	LocalStatusClean    = "clean"
	LocalStatusModified = "modified"

	RemoteStatusBehind      = "behind"
	RemoteStatusPinOutdated = "pin_outdated"
	RemoteStatusUpToDate    = "up_to_date"

	PullStatusPulled          = "pulled"
	PullStatusAlreadyUpToDate = "already_up_to_date"

	FetchStatusUpdatesAvailable = "updates_available"

	PushStatusPushed    = "pushed"
	PushStatusNoChanges = "no_changes"

	PushReasonLocalEdit  = "local_edit"
	PushReasonRefCascade = "ref_cascade"
)

func PullHint(owner, name string) string {
	return fmt.Sprintf("run 'clier pull %s/%s'", owner, name)
}

func PinOutdatedHint() string {
	return "team still pins this version; edit team to upgrade"
}
