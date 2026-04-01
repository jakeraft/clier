package dashboard

import "time"

type DashboardData struct {
	Teams         []TeamView         `json:"teams"`
	Members       []MemberView       `json:"members"`
	Sprints       []SprintView       `json:"sprints"`
	CliProfiles   []CliProfileView   `json:"cliProfiles"`
	SystemPrompts []SystemPromptView `json:"systemPrompts"`
	GitRepos      []GitRepoView      `json:"gitRepos"`
}

type TeamView struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	RootMemberID   string         `json:"rootMemberId"`
	MemberIDs      []string       `json:"memberIds"`
	Relations      []RelationView `json:"relations"`
	RootMemberName string         `json:"rootMemberName"`
	MemberNames    []string       `json:"memberNames"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
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
	GitRepoID         *string   `json:"gitRepoId"`
	CliProfileName    string    `json:"cliProfileName"`
	SystemPromptNames []string  `json:"systemPromptNames"`
	GitRepoName       *string   `json:"gitRepoName"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type SprintView struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	TeamSnapshot TeamSnapshotView `json:"teamSnapshot"`
	State        string           `json:"state"`
	Error        *string          `json:"error"`
	TeamName     string           `json:"teamName"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

type TeamSnapshotView struct {
	TeamName     string               `json:"teamName"`
	RootMemberID string               `json:"rootMemberId"`
	Members      []MemberSnapshotView `json:"members"`
}

type MemberSnapshotView struct {
	MemberID       string               `json:"memberId"`
	MemberName     string               `json:"memberName"`
	Binary         string               `json:"binary"`
	Model          string               `json:"model"`
	CliProfileName string               `json:"cliProfileName"`
	SystemArgs     []string             `json:"systemArgs"`
	CustomArgs     []string             `json:"customArgs"`
	DotConfig      map[string]any       `json:"dotConfig"`
	SystemPrompts  []PromptSnapshotView `json:"systemPrompts"`
	GitRepo        *GitRepoSnapshotView `json:"gitRepo"`
	Relations      RelationsView        `json:"relations"`
}

type PromptSnapshotView struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type GitRepoSnapshotView struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RelationsView struct {
	Leaders []string `json:"leaders"`
	Workers []string `json:"workers"`
	Peers   []string `json:"peers"`
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
	Bundled   bool      `json:"bundled"`
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
