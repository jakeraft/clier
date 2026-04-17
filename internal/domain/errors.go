package domain

import "maps"

// Kind classifies a Fault. Each Kind maps 1:1 to a single user-facing
// template in the message catalog.
//
// Adding a new Kind requires:
//  1. add the constant here
//  2. add it to AllKinds()
//  3. add a template entry in cmd/messages/catalog.go
//  4. (if it originates from an adapter) add a translation in
//     internal/app/translate.go
//
// The catalog and translation tests fail when any of these are missing.
type Kind string

const (
	// Authentication & authorization.
	KindAuthRequired      Kind = "auth_required"
	KindAuthFailed        Kind = "auth_failed"
	KindInvalidOAuthState Kind = "invalid_oauth_state"
	KindForbidden         Kind = "forbidden"
	KindNotOrgMember      Kind = "not_org_member"
	KindNotOrgOwner       Kind = "not_org_owner"

	// Lookup failures originating from the server.
	KindUserNotFound            Kind = "user_not_found"
	KindOrgNotFound             Kind = "org_not_found"
	KindResourceNotFound        Kind = "resource_not_found"
	KindResourceVersionNotFound Kind = "resource_version_not_found"
	KindOrgMemberNotFound       Kind = "org_member_not_found"
	KindTokenNotFound           Kind = "token_not_found"

	// Conflict & validation from the server.
	KindResourceNameTaken Kind = "resource_name_taken"
	KindOrgMemberExists   Kind = "org_member_exists"
	KindInvalidArgument   Kind = "invalid_argument"
	KindUnknownCommand    Kind = "unknown_command"
	KindNotTeamResource   Kind = "not_team_resource"

	// Local workspace / runtime faults.
	KindCloneDestExists         Kind = "clone_dest_exists"
	KindRunNotFound             Kind = "run_not_found"
	KindRunInactive             Kind = "run_inactive"
	KindRunAlreadyRunning       Kind = "run_already_running"
	KindRunBlocksRemove         Kind = "run_blocks_remove"
	KindWorkspaceDirty          Kind = "workspace_dirty"
	KindWorkingCopyMissing      Kind = "working_copy_missing"
	KindWorkingCopyIncomplete   Kind = "working_copy_incomplete"
	KindInvalidResourceRef      Kind = "invalid_resource_ref"
	KindUnsupportedKind         Kind = "unsupported_kind"
	KindUnsupportedPlatform     Kind = "unsupported_platform"
	KindAuthTimeout             Kind = "auth_timeout"
	KindContentRequired         Kind = "content_required"
	KindRunIDRequired           Kind = "run_id_required"
	KindWorkspaceDirNotAbsolute Kind = "workspace_dir_not_absolute"
	KindOwnerRequired           Kind = "owner_required"
	KindLoginRequired           Kind = "login_required"
	KindRemoteChanged           Kind = "remote_changed"
	KindManifestIncompatible    Kind = "manifest_incompatible"
	KindRepoDirConflict         Kind = "repo_dir_conflict"
	KindRunStartTimeout         Kind = "run_start_timeout"
	KindNotATerminal            Kind = "not_a_terminal"
	KindMissingArgument         Kind = "missing_argument"
	KindTooManyArgs             Kind = "too_many_args"

	// Transport / unknown.
	KindServerUnreachable Kind = "server_unreachable"
	KindInternal          Kind = "internal"
)

// AllKinds returns every Kind. Used by catalog completeness tests.
func AllKinds() []Kind {
	return []Kind{
		KindAuthRequired,
		KindAuthFailed,
		KindInvalidOAuthState,
		KindForbidden,
		KindNotOrgMember,
		KindNotOrgOwner,
		KindUserNotFound,
		KindOrgNotFound,
		KindResourceNotFound,
		KindResourceVersionNotFound,
		KindOrgMemberNotFound,
		KindTokenNotFound,
		KindResourceNameTaken,
		KindOrgMemberExists,
		KindInvalidArgument,
		KindUnknownCommand,
		KindNotTeamResource,
		KindCloneDestExists,
		KindRunNotFound,
		KindRunInactive,
		KindRunAlreadyRunning,
		KindRunBlocksRemove,
		KindWorkspaceDirty,
		KindWorkingCopyMissing,
		KindWorkingCopyIncomplete,
		KindInvalidResourceRef,
		KindUnsupportedKind,
		KindUnsupportedPlatform,
		KindAuthTimeout,
		KindContentRequired,
		KindRunIDRequired,
		KindWorkspaceDirNotAbsolute,
		KindOwnerRequired,
		KindLoginRequired,
		KindRemoteChanged,
		KindManifestIncompatible,
		KindRepoDirConflict,
		KindRunStartTimeout,
		KindNotATerminal,
		KindMissingArgument,
		KindTooManyArgs,
		KindServerUnreachable,
		KindInternal,
	}
}

// Fault is the domain-level error value. Adapters and use cases return
// Fault (or wrap into one via app.Translate); the CLI presenter renders
// it via the message catalog. Fault.Error() is debug-only — never shown
// to end users without going through the catalog.
type Fault struct {
	Kind    Kind
	Subject map[string]string // owner, name, run_id, path, version, ...
	Cause   error             // optional infra detail for logs
}

func (f *Fault) Error() string { return string(f.Kind) }

func (f *Fault) Unwrap() error { return f.Cause }

// Newf constructs a Fault with no subject.
func Newf(kind Kind, cause error) *Fault {
	return &Fault{Kind: kind, Cause: cause}
}

// With returns a copy of the fault with the given subject keys merged in.
func (f *Fault) With(kv ...string) *Fault {
	if len(kv)%2 != 0 {
		panic("domain.Fault.With requires key/value pairs")
	}
	cp := &Fault{Kind: f.Kind, Cause: f.Cause}
	if len(f.Subject) > 0 || len(kv) > 0 {
		cp.Subject = make(map[string]string, len(f.Subject)+len(kv)/2)
		maps.Copy(cp.Subject, f.Subject)
		for i := 0; i < len(kv); i += 2 {
			cp.Subject[kv[i]] = kv[i+1]
		}
	}
	return cp
}
