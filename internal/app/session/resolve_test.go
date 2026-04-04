package session

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestResolvePlaceholders(t *testing.T) {
	m := domain.MemberPlan{
		Terminal: domain.TerminalPlan{
			Command: "export CLIER_SESSION_ID='{{CLIER_SESSION_ID}}' && export CLAUDE_CODE_OAUTH_TOKEN='{{CLIER_AUTH_CLAUDE}}' && cd '{{CLIER_MEMBERSPACE}}/project'",
		},
		Workspace: domain.WorkspacePlan{
			Memberspace: "{{CLIER_BASE}}/workspaces/{{CLIER_SESSION_ID}}/member1",
			Files: []domain.FileEntry{
				{
					Path:    "{{CLIER_MEMBERSPACE}}/.claude/settings.json",
					Content: `{"excludes":["~/.claude/**"],"projects":{"{{CLIER_MEMBERSPACE}}/project":{}}}`,
				},
			},
		},
	}

	resolved := resolvePlaceholders(m, "/home/user/.clier", "/home/user", "session-999", "sk-token-123")

	if resolved.Workspace.Memberspace != "/home/user/.clier/workspaces/session-999/member1" {
		t.Errorf("Memberspace = %q", resolved.Workspace.Memberspace)
	}

	wantCmd := "export CLIER_SESSION_ID='session-999' && export CLAUDE_CODE_OAUTH_TOKEN='sk-token-123' && cd '/home/user/.clier/workspaces/session-999/member1/project'"
	if resolved.Terminal.Command != wantCmd {
		t.Errorf("Command = %q\nwant    %q", resolved.Terminal.Command, wantCmd)
	}

	if resolved.Workspace.Files[0].Path != "/home/user/.clier/workspaces/session-999/member1/.claude/settings.json" {
		t.Errorf("Files[0].Path = %q", resolved.Workspace.Files[0].Path)
	}

	// ~/ should be expanded to home dir, placeholders should be resolved
	wantContent := `{"excludes":["/home/user/.claude/**"],"projects":{"/home/user/.clier/workspaces/session-999/member1/project":{}}}`
	if resolved.Workspace.Files[0].Content != wantContent {
		t.Errorf("Files[0].Content = %q\nwant             %q", resolved.Workspace.Files[0].Content, wantContent)
	}
}
