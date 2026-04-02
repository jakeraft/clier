package sprint

import (
	"fmt"
	"path/filepath"

	"github.com/jakeraft/clier/internal/domain"
)

// BuildSprintSnapshot transforms a TeamSnapshot into a fully resolved SprintSnapshot.
// Pure function — no I/O. All paths, prompts, envs are resolved into the snapshot.
func BuildSprintSnapshot(sprintID, baseDir string, team domain.TeamSnapshot) (domain.SprintSnapshot, error) {
	members := make([]domain.SprintMemberSnapshot, 0, len(team.Members))

	for _, m := range team.Members {
		home := filepath.Join(baseDir, sprintID, m.MemberID)
		workDir := filepath.Join(home, "project")

		prompt, err := BuildMemberPrompt(team, m.MemberID)
		if err != nil {
			return domain.SprintSnapshot{}, fmt.Errorf("build prompt for %s: %w", m.MemberName, err)
		}

		cmd, err := BuildCommand(m, prompt, workDir, sprintID, home)
		if err != nil {
			return domain.SprintSnapshot{}, fmt.Errorf("build command for %s: %w", m.MemberName, err)
		}

		members = append(members, domain.SprintMemberSnapshot{
			MemberID:   m.MemberID,
			MemberName: m.MemberName,
			Relations:  m.Relations,
			Home:       home,
			WorkDir:    workDir,
			Binary:     m.Binary,
			DotConfig:  m.DotConfig,
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
