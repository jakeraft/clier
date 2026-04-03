package dashboard

import "time"

type DashboardData struct {
	Teams         []TeamView         `json:"teams"`
	Members       []MemberView       `json:"members"`
	CliProfiles   []CliProfileView   `json:"cliProfiles"`
	SystemPrompts []SystemPromptView `json:"systemPrompts"`
	GitRepos      []GitRepoView      `json:"gitRepos"`
	Envs          []EnvView          `json:"envs"`
}

type TeamView struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	RootMemberID   string              `json:"rootMemberId"`
	MemberIDs      []string            `json:"memberIds"`
	Relations      []RelationView      `json:"relations"`
	Plan           []MemberSessionPlanView `json:"plan"`
	RootMemberName string              `json:"rootMemberName"`
	MemberNames    []string            `json:"memberNames"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
}

type MemberSessionPlanView struct {
	MemberID    string          `json:"memberId"`
	MemberName  string          `json:"memberName"`
	Memberspace string          `json:"memberspace"`
	Command     string          `json:"command"`
	GitRepo     *GitRepoRef     `json:"gitRepo"`
	Files       []FileEntryView `json:"files"`
}

type GitRepoRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type FileEntryView struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RelationView struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type MemberView struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	CliProfileID      string    `json:"cliProfileId"`
	SystemPromptIDs   []string  `json:"systemPromptIds"`
	EnvIDs            []string  `json:"envIds"`
	GitRepoID         *string   `json:"gitRepoId"`
	CliProfileName    string    `json:"cliProfileName"`
	SystemPromptNames []string  `json:"systemPromptNames"`
	EnvNames          []string  `json:"envNames"`
	GitRepoName       *string   `json:"gitRepoName"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type CliProfileView struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Model      string         `json:"model"`
	Binary     string         `json:"binary"`
	SystemArgs []string       `json:"systemArgs"`
	CustomArgs []string       `json:"customArgs"`
	DotConfig  map[string]any `json:"dotConfig"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type SystemPromptView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prompt    string    `json:"prompt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GitRepoView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type EnvView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}


