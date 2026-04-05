package task

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestExpandPlaceholders(t *testing.T) {
	m := domain.MemberPlan{
		Terminal: domain.TerminalPlan{
			Command: "export CLIER_TASK_ID='{{CLIER_TASK_ID}}' && export CLAUDE_CODE_OAUTH_TOKEN='{{CLIER_AUTH_CLAUDE}}' && cd '{{CLIER_MEMBERSPACE}}/project'",
		},
		Workspace: domain.WorkspacePlan{
			Memberspace: "{{CLIER_BASE}}/workspaces/{{CLIER_TASK_ID}}/member1",
			Files: []domain.FileEntry{
				{
					Path:    "{{CLIER_MEMBERSPACE}}/.claude/settings.json",
					Content: `{"excludes":["~/.claude/**"],"projects":{"{{CLIER_MEMBERSPACE}}/project":{}}}`,
				},
			},
		},
	}

	expanded := expandPlaceholders(m, "/home/user/.clier", "/home/user", "task-999", "sk-token-123")

	if expanded.Workspace.Memberspace != "/home/user/.clier/workspaces/task-999/member1" {
		t.Errorf("Memberspace = %q", expanded.Workspace.Memberspace)
	}

	wantCmd := "export CLIER_TASK_ID='task-999' && export CLAUDE_CODE_OAUTH_TOKEN='sk-token-123' && cd '/home/user/.clier/workspaces/task-999/member1/project'"
	if expanded.Terminal.Command != wantCmd {
		t.Errorf("Command = %q\nwant    %q", expanded.Terminal.Command, wantCmd)
	}

	if expanded.Workspace.Files[0].Path != "/home/user/.clier/workspaces/task-999/member1/.claude/settings.json" {
		t.Errorf("Files[0].Path = %q", expanded.Workspace.Files[0].Path)
	}

	// ~/ should be expanded to home dir, placeholders should be expanded
	wantContent := `{"excludes":["/home/user/.claude/**"],"projects":{"/home/user/.clier/workspaces/task-999/member1/project":{}}}`
	if expanded.Workspace.Files[0].Content != wantContent {
		t.Errorf("Files[0].Content = %q\nwant             %q", expanded.Workspace.Files[0].Content, wantContent)
	}
}
