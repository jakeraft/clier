package domain

type TeamSnapshot struct {
	TeamName     string           `json:"team_name"`
	RootMemberID string           `json:"root_member_id"`
	Members      []MemberSnapshot `json:"members"`
}

type MemberSnapshot struct {
	MemberID       string           `json:"member_id"`
	MemberName     string           `json:"member_name"`
	Binary         CliBinary        `json:"binary"`
	Model          string           `json:"model"`
	CliProfileName string           `json:"cli_profile_name"`
	SystemArgs     []string         `json:"system_args"`
	CustomArgs     []string         `json:"custom_args"`
	DotConfig      DotConfig        `json:"dot_config"`
	SystemPrompts  []PromptSnapshot `json:"system_prompts"`
	GitRepo        *GitRepoSnapshot `json:"git_repo"` // nil means not set
	Envs           []EnvSnapshot    `json:"envs"`
	Relations      MemberRelations  `json:"relations"`
}

type PromptSnapshot struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type GitRepoSnapshot struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EnvSnapshot struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// FindMember returns the member with the given ID, or false if not found.
func (s TeamSnapshot) FindMember(id string) (MemberSnapshot, bool) {
	for _, m := range s.Members {
		if m.MemberID == id {
			return m, true
		}
	}
	return MemberSnapshot{}, false
}

// MemberName returns the name of the member with the given ID, or empty string.
func (s TeamSnapshot) MemberName(id string) string {
	if m, ok := s.FindMember(id); ok {
		return m.MemberName
	}
	return ""
}

// IsConnected returns true if fromID has a relation to toID.
func (s TeamSnapshot) IsConnected(fromID, toID string) bool {
	from, ok := s.FindMember(fromID)
	if !ok {
		return false
	}
	return from.Relations.IsConnectedTo(toID)
}

// FileEntry is a resolved config file to write to a member's workspace.
type FileEntry struct {
	Path    string `json:"path"`    // relative to member Home
	Content string `json:"content"`
}

// SprintSnapshot is the resolved execution plan stored in a Sprint.
// Built from TeamSnapshot by the sprint service.
type SprintSnapshot struct {
	TeamName     string                 `json:"team_name"`
	RootMemberID string                 `json:"root_member_id"`
	Members      []SprintMemberSnapshot `json:"members"`
}

// SprintMemberSnapshot is a fully resolved member execution plan.
type SprintMemberSnapshot struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`

	Home    string           `json:"home"`
	WorkDir string           `json:"work_dir"`
	Files   []FileEntry      `json:"files"`
	GitRepo *GitRepoSnapshot `json:"git_repo"`

	Command string `json:"command"`
}
