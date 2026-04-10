package cmd

import "fmt"

const (
	resourceKindMember         = "member"
	resourceKindTeam           = "team"
	resourceKindClaudeMd       = "claude-md"
	resourceKindClaudeSettings = "claude-settings"
	resourceKindSkill          = "skill"
)

func errUnsupportedResourceKind(kind string) error {
	return fmt.Errorf("unsupported resource kind %q", kind)
}
