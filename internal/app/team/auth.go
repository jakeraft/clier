package team

import "github.com/jakeraft/clier/internal/domain"

// AuthResult holds auth-related artifacts for a single binary type.
// Claude uses env-var based auth (CommandEnvs); Codex uses a file (Files).
type AuthResult struct {
	CommandEnvs []string           // e.g. ["CLAUDE_CODE_OAUTH_TOKEN={{CLIER_AUTH_CLAUDE}}"]
	Files       []domain.FileEntry // e.g. [{".codex/auth.json", "{{CLIER_AUTH_CODEX}}"}]
}

// setAuth returns auth placeholders appropriate for the given binary.
func setAuth(binary domain.CliBinary) AuthResult {
	switch binary {
	case domain.BinaryClaude:
		return AuthResult{
			CommandEnvs: []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude},
		}
	case domain.BinaryCodex:
		return AuthResult{
			Files: []domain.FileEntry{
				{
					Path:    PlaceholderMemberspace + "/.codex/auth.json",
					Content: PlaceholderAuthCodex,
				},
			},
		}
	default:
		return AuthResult{}
	}
}
