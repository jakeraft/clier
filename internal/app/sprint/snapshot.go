package sprint

import (
	"fmt"
	"path/filepath"

	"github.com/jakeraft/clier/internal/domain"
)

// BuildSprintSnapshot transforms a TeamSnapshot into a fully resolved SprintSnapshot.
// Pure function — no I/O. All paths, prompts, envs, configs are resolved into the snapshot.
func BuildSprintSnapshot(sprintID, baseDir string, team domain.TeamSnapshot, tokens map[domain.CliBinary]string) (domain.SprintSnapshot, error) {
	members := make([]domain.SprintMemberSnapshot, 0, len(team.Members))

	for _, m := range team.Members {
		home := filepath.Join(baseDir, sprintID, m.MemberID)
		workDir := filepath.Join(home, "project")

		prompt, err := BuildMemberPrompt(team, m.MemberID)
		if err != nil {
			return domain.SprintSnapshot{}, fmt.Errorf("build prompt for %s: %w", m.MemberName, err)
		}

		token := tokens[m.Binary]
		cmd, err := BuildCommand(m, prompt, workDir, sprintID, home, token)
		if err != nil {
			return domain.SprintSnapshot{}, fmt.Errorf("build command for %s: %w", m.MemberName, err)
		}

		var files []domain.FileEntry
		switch m.Binary {
		case domain.BinaryClaude:
			files = buildClaudeFiles(m.DotConfig, workDir)
		case domain.BinaryCodex:
			files = buildCodexFiles(m.DotConfig, workDir)
		}

		members = append(members, domain.SprintMemberSnapshot{
			MemberID:   m.MemberID,
			MemberName: m.MemberName,
			Home:       home,
			WorkDir:    workDir,
			Files:      files,
			GitRepo:    m.GitRepo,
			Command:    cmd,
		})
	}

	return domain.SprintSnapshot{
		TeamName:     team.TeamName,
		RootMemberID: team.RootMemberID,
		Members:      members,
	}, nil
}
