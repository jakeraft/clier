package tutorial

import "github.com/jakeraft/clier/internal/domain"

func init() {
	Register(StoryPair)
}

var StoryPair = &Scenario{
	Name:        "story-pair",
	Description: "Simple 2-member team: editor delegates to one writer",

	SystemPrompts: []SystemPromptDef{
		{
			Name: "pair-editor",
			Prompt: `You are an Editor. Plan a very short story (3-5 sentences) and delegate writing to your worker.

Process:
1. Plan a brief story premise (genre, character, conflict).
2. Send the premise to your worker and ask them to write it.
3. Wait for the completed story from your worker.
4. Review and send the final result back.`,
		},
		{
			Name: "pair-writer",
			Prompt: `You are a Writer. Write a very short story from the brief you receive.

Rules:
- Write exactly 3-5 sentences.
- Each sentence max 10 words.
- Be vivid but concise.

When finished, send the full story text to your leader.`,
		},
	},

	GitRepos: []GitRepoDef{
		{Name: "pair-repo", URL: "https://github.com/jakeraft/clier_hello.git"},
	},

	CliProfiles: []CliProfileDef{
		{Name: "pair-sonnet", PresetKey: "claude-sonnet"},
	},

	Members: []MemberDef{
		{Name: "editor", CliProfileName: "pair-sonnet", SystemPromptNames: []string{"pair-editor"}, GitRepoName: "pair-repo"},
		{Name: "writer", CliProfileName: "pair-sonnet", SystemPromptNames: []string{"pair-writer"}, GitRepoName: "pair-repo"},
	},

	Team: TeamDef{
		Name:           "story-pair",
		RootMemberName: "editor",
	},

	Relations: []RelationDef{
		{From: "editor", To: "writer", Type: domain.RelationLeader},
	},
}
