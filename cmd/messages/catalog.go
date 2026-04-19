// Package messages is the single source of truth for user-facing
// CLI text. Every domain.Kind maps to exactly one Template here.
//
// Domain code, adapters, and command handlers MUST NOT build user
// messages directly — they construct domain.Fault values, and the
// CLI presenter renders them through Render.
package messages

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// Template defines how a Fault renders into user-facing text.
//
// Format and HintFormat each declare their own Subject keys via
// FormatArgs and HintArgs so the two strings can use different
// placeholder counts without affecting each other. Missing Subject
// values render as "<unknown>".
type Template struct {
	Format     string
	FormatArgs []string
	HintFormat string
	HintArgs   []string
}

// Rendered is the result of applying a Template to a Fault.
type Rendered struct {
	Message string
	Hint    string
}

// catalog maps every domain.Kind to its template. Completeness is
// verified by TestEveryKindHasTemplate.
var catalog = map[domain.Kind]Template{
	domain.KindAuthRequired: {
		Format:     "authentication required",
		HintFormat: "run 'clier auth login' to sign in",
	},
	domain.KindAuthFailed: {
		Format:     "authentication failed",
		HintFormat: "run 'clier auth login' to refresh credentials",
	},
	domain.KindInvalidOAuthState: {
		Format:     "OAuth state is invalid or expired",
		HintFormat: "run 'clier auth login' again",
	},
	domain.KindForbidden: {
		Format: "you do not have permission to perform this action",
	},
	domain.KindNotOrgMember: {
		Format: "you are not a member of this organization",
	},
	domain.KindNotOrgOwner: {
		Format: "this action requires organization owner permissions",
	},

	domain.KindUserNotFound: {
		Format:     "user %q does not exist",
		FormatArgs: []string{"owner"},
	},
	domain.KindOrgNotFound: {
		Format:     "organization %q does not exist",
		FormatArgs: []string{"owner"},
	},
	domain.KindResourceNotFound: {
		Format:     "resource %s/%s does not exist",
		FormatArgs: []string{"owner", "name"},
	},
	domain.KindResourceVersionNotFound: {
		Format:     "resource %s/%s has no version %s",
		FormatArgs: []string{"owner", "name", "version"},
	},
	domain.KindOrgMemberNotFound: {
		Format: "the user is not a member of this organization",
	},
	domain.KindTokenNotFound: {
		Format:     "personal access token not found",
		HintFormat: "run 'clier auth login' to refresh credentials",
	},

	domain.KindResourceNameTaken: {
		Format:     "resource %s/%s already exists",
		FormatArgs: []string{"owner", "name"},
		HintFormat: "choose a different name or fork it",
	},
	domain.KindOrgMemberExists: {
		Format: "the user is already a member of this organization",
	},
	domain.KindInvalidArgument: {
		Format:     "invalid request: %s",
		FormatArgs: []string{"detail"},
	},
	domain.KindUnknownCommand: {
		Format:     "unknown command: %q",
		FormatArgs: []string{"command"},
		HintFormat: "run 'clier --help' to see available commands",
	},
	domain.KindNotTeamResource: {
		Format: "this resource is not a team",
	},

	domain.KindCloneDestExists: {
		Format:     "clone destination already exists: %s",
		FormatArgs: []string{"path"},
		HintFormat: "remove with 'clier remove %s/%s' or pull updates with 'clier pull %s/%s'",
		HintArgs:   []string{"owner", "name", "owner", "name"},
	},
	domain.KindRunNotFound: {
		Format:     "run %s not found",
		FormatArgs: []string{"run_id"},
		HintFormat: "list active runs with 'clier run list'",
	},
	domain.KindRunInactive: {
		Format:     "the run is no longer active",
		HintFormat: "start it again with 'clier run start <owner/name>'",
	},
	domain.KindRunAlreadyRunning: {
		Format:     "run %s is already running for this working copy",
		FormatArgs: []string{"run_id"},
		HintFormat: "stop it first with 'clier run stop %s', or fork the team to run in parallel",
		HintArgs:   []string{"run_id"},
	},
	domain.KindRunBlocksRemove: {
		Format:     "run %s is still running",
		FormatArgs: []string{"run_id"},
		HintFormat: "stop it first with 'clier run stop %s' before removing",
		HintArgs:   []string{"run_id"},
	},
	domain.KindWorkspaceDirty: {
		Format:     "working copy has uncommitted changes (%s modified)",
		FormatArgs: []string{"modified"},
		HintFormat: "push or revert before removing",
	},
	domain.KindPullBlockedDirty: {
		Format:     "working copy has uncommitted changes (%s modified)",
		FormatArgs: []string{"modified"},
		HintFormat: "push or revert before pulling",
	},
	domain.KindWorkingCopyMissing: {
		Format:     "no working copy at %s",
		FormatArgs: []string{"path"},
		HintFormat: "run 'clier clone %s/%s' first",
		HintArgs:   []string{"owner", "name"},
	},
	domain.KindWorkingCopyIncomplete: {
		Format:     "local clone is incomplete: %s",
		FormatArgs: []string{"detail"},
		HintFormat: "run 'clier pull <owner/name>' to refresh the working copy",
	},
	domain.KindInvalidResourceRef: {
		Format:     "invalid resource ref %q",
		FormatArgs: []string{"ref"},
		HintFormat: "use the form <owner/name> or <owner/name>@<version>",
	},
	domain.KindUnsupportedKind: {
		Format:     "unsupported resource kind %q",
		FormatArgs: []string{"resource_kind"},
	},
	domain.KindUnsupportedPlatform: {
		Format:     "unsupported platform %s",
		FormatArgs: []string{"platform"},
	},
	domain.KindAuthTimeout: {
		Format:     "login timed out",
		HintFormat: "run 'clier auth login' to try again",
	},
	domain.KindContentRequired: {
		Format:     "no content provided",
		HintFormat: "pass content as an argument or pipe it via stdin",
	},
	domain.KindRunIDRequired: {
		Format:     "run id required",
		HintFormat: "pass --run <run-id> or set the %s environment variable",
		HintArgs:   []string{"env"},
	},
	domain.KindWorkspaceDirNotAbsolute: {
		Format:     "workspace_dir must be an absolute path: %q",
		FormatArgs: []string{"path"},
	},
	domain.KindOwnerRequired: {
		Format:     "owner is required",
		HintFormat: "pass --owner <name> or run 'clier auth login' to set a default",
	},
	domain.KindLoginRequired: {
		Format:     "this command requires login",
		HintFormat: "run 'clier auth login' to sign in",
	},
	domain.KindAgentNotInRun: {
		Format:     "agent %s is not in the current run plan",
		FormatArgs: []string{"agent"},
		HintFormat: "run 'clier run view <run-id>' to see available agents",
	},
	domain.KindRemoteChanged: {
		Format:     "remote %s %s/%s has changed",
		FormatArgs: []string{"resource_kind", "owner", "name"},
		HintFormat: "pull before pushing: 'clier pull %s'",
		HintArgs:   []string{"workspace"},
	},
	domain.KindManifestIncompatible: {
		Format:     "local clone manifest is incompatible (format %s, expected %s)",
		FormatArgs: []string{"got", "expected"},
		HintFormat: "%s",
		HintArgs:   []string{"hint"},
	},
	domain.KindRepoDirConflict: {
		Format:     "repo directory %s is unusable: %s",
		FormatArgs: []string{"path", "detail"},
		HintFormat: "remove the directory or choose a different workspace_dir",
	},
	domain.KindRunStartTimeout: {
		Format:     "agent %s did not become ready in time",
		FormatArgs: []string{"agent"},
		HintFormat: "check the agent's terminal with 'clier run attach <run-id>' or restart with 'clier run start'",
	},
	domain.KindNotATerminal: {
		Format:     "this command needs a real terminal",
		HintFormat: "run it from a normal user terminal (not inside an agent or pipe)",
	},
	domain.KindMissingArgument: {
		Format:     "missing argument",
		HintFormat: "usage: %s",
		HintArgs:   []string{"usage"},
	},
	domain.KindTooManyArgs: {
		Format:     "too many arguments",
		HintFormat: "usage: %s",
		HintArgs:   []string{"usage"},
	},

	domain.KindServerUnreachable: {
		Format:     "cannot reach clier server (connection refused)",
		HintFormat: "start the server, or check 'server_url' in ~/.clier/config.json",
	},
	domain.KindInternal: {
		Format:     "the server encountered an internal error",
		HintFormat: "try again or check the server logs",
	},
}

// Render applies the catalog Template for f.Kind. An unknown Kind falls
// back to a debug-shaped message so missing entries are visible without
// crashing the CLI.
func Render(f *domain.Fault) Rendered {
	if f == nil {
		return Rendered{Message: "unknown error"}
	}
	t, ok := catalog[f.Kind]
	if !ok {
		return Rendered{Message: fmt.Sprintf("unhandled error kind: %s", f.Kind)}
	}
	return Rendered{
		Message: format(t.Format, lookupArgs(f.Subject, t.FormatArgs)),
		Hint:    format(t.HintFormat, lookupArgs(f.Subject, t.HintArgs)),
	}
}

func lookupArgs(subject map[string]string, keys []string) []any {
	if len(keys) == 0 {
		return nil
	}
	out := make([]any, len(keys))
	for i, k := range keys {
		if v, ok := subject[k]; ok && v != "" {
			out[i] = v
		} else {
			out[i] = "<unknown>"
		}
	}
	return out
}

func format(fmtStr string, args []any) string {
	if fmtStr == "" {
		return ""
	}
	if len(args) == 0 {
		return fmtStr
	}
	return strings.TrimSpace(fmt.Sprintf(fmtStr, args...))
}

// HasTemplate reports whether kind is registered. Used by tests.
func HasTemplate(kind domain.Kind) bool {
	_, ok := catalog[kind]
	return ok
}
