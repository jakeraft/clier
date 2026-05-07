package main

import "github.com/jakeraft/clier/cmd"

// Build-time identity. All three are -ldflags overridable.
//
//	version  — semver tag for release builds, "dev" otherwise.
//	channel  — "prod" (brew, source) or "dev" (make install-dev).
//	           Used by `clier version` so QA agents can sanity-check
//	           which binary they are talking to.
//	commit   — short git sha, optional.
var (
	version = "dev"
	channel = "prod"
	commit  = ""
)

func main() {
	cmd.SetBuildInfo(version, channel, commit)
	cmd.Execute()
}
