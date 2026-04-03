package runplan

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

const (
	PlaceholderBase        = "{{CLIER_BASE}}"
	PlaceholderMemberspace = "{{CLIER_MEMBERSPACE}}"
	PlaceholderSessionID   = "{{CLIER_SESSION_ID}}"
	PlaceholderAuthClaude  = "{{CLIER_AUTH_CLAUDE}}"
	PlaceholderAuthCodex   = "{{CLIER_AUTH_CODEX}}"
)

// TeamData is the aggregate team state needed to build a plan.
type TeamData struct {
	TeamID       string
	TeamName     string
	RootMemberID string
	Members      []MemberData
}

// MemberData is the aggregate member state needed to build a MemberSessionPlan.
type MemberData struct {
	MemberID      string
	MemberName    string
	Binary        domain.CliBinary
	Model         string
	SystemArgs    []string
	CustomArgs    []string
	DotConfig     domain.DotConfig
	SystemPrompts []domain.PromptSnapshot
	GitRepo       *domain.GitRepoSnapshot
	Envs          []domain.EnvSnapshot
	Relations     domain.MemberRelations
}

// BuildPlan builds a complete execution plan from team data.
// All machine-specific values use placeholders. The teamID is used as workspace namespace.
func BuildPlan(td TeamData) ([]domain.MemberSessionPlan, error) {
	nameByID := make(map[string]string, len(td.Members))
	for _, m := range td.Members {
		nameByID[m.MemberID] = m.MemberName
	}

	members := make([]domain.MemberSessionPlan, 0, len(td.Members))

	for _, m := range td.Members {
		memberspace := fmt.Sprintf("%s/%s/%s", PlaceholderBase, td.TeamID, m.MemberID)

		clierPrompt := buildClierPrompt(td.TeamName, m, nameByID)
		userPrompt := joinPrompts(m.SystemPrompts)
		prompt := "---\n\n" + clierPrompt + "\n---\n\n" + userPrompt

		auth := setAuth(m.Binary)

		files, err := buildFiles(m.Binary, m.DotConfig, PlaceholderMemberspace)
		if err != nil {
			return nil, fmt.Errorf("build files for %s: %w", m.MemberName, err)
		}
		files = append(files, auth.Files...)

		cmd, err := buildCommand(
			m.Binary, m.Model, m.SystemArgs, m.CustomArgs,
			prompt, PlaceholderSessionID, m.MemberID,
			auth.CommandEnvs, m.Envs,
		)
		if err != nil {
			return nil, fmt.Errorf("build command for %s: %w", m.MemberName, err)
		}

		var gitRepo *domain.GitRepoRef
		if m.GitRepo != nil {
			gitRepo = &domain.GitRepoRef{
				Name: m.GitRepo.Name,
				URL:  m.GitRepo.URL,
			}
		}

		launchPath := PlaceholderMemberspace + "/launch.sh"
		files = append(files, domain.FileEntry{Path: launchPath, Content: cmd})

		members = append(members, domain.MemberSessionPlan{
			MemberID:   m.MemberID,
			MemberName: m.MemberName,
			Terminal:   domain.TerminalPlan{Command: ". " + launchPath},
			Workspace: domain.WorkspacePlan{
				Memberspace: memberspace,
				Files:       files,
				GitRepo:     gitRepo,
			},
		})
	}

	return members, nil
}

// buildFiles dispatches to the binary-specific config file builder.
func buildFiles(binary domain.CliBinary, dotConfig domain.DotConfig,
	memberspacePlaceholder string) ([]domain.FileEntry, error) {

	workDir := memberspacePlaceholder + "/project"

	switch binary {
	case domain.BinaryClaude:
		return buildClaudeFiles(dotConfig, workDir, memberspacePlaceholder)
	case domain.BinaryCodex:
		return buildCodexFiles(dotConfig, workDir, memberspacePlaceholder)
	default:
		return nil, fmt.Errorf("unknown binary: %s", binary)
	}
}

