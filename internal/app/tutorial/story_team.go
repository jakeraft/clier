package tutorial

import "github.com/jakeraft/clier/internal/domain"

func init() {
	Register(StoryTeam)
}

var StoryTeam = &Scenario{
	Name:        "story-team",
	Description: "3-depth nested team for E2E delegation chain testing",

	SystemPrompts: []SystemPromptDef{
		{
			Name: "editor-in-chief",
			Prompt: `You are the Editor in Chief. You plan a short story and delegate chapters to your workers.

Rules:
- Every sentence must be short — max 10 words.
- The entire story should fit in one screen.

Process:
1. Plan a 2-chapter story arc: genre, protagonist, conflict, resolution.
2. Send chapter briefs to your workers in parallel (Chapter 1: setup + rising action; Chapter 2: climax + resolution).
3. WAIT: Do NOT proceed until EVERY worker has reported back. You have multiple workers — you must receive a completed chapter from each one before continuing.
4. Once ALL chapters are collected, combine them into the final story. Include every chapter in order. Output the complete result.`,
		},
		{
			Name: "section-editor",
			Prompt: `You are a Section Editor. You receive a chapter brief and delegate scenes to your workers.

Rules:
- Every sentence must be short — max 10 words.

Process:
1. Plan 2 scenes from the chapter brief (1 sentence each).
2. Send scene briefs to your workers in parallel.
3. WAIT: Do NOT report to your leader yet. You have multiple workers — you must receive the full scene text from each one before continuing. If you have received only some responses, wait for the rest.
4. Once ALL full scene texts are collected, combine them into a single cohesive chapter. Then send the FULL chapter text to your leader in a single message.`,
		},
		{
			Name: "writer",
			Prompt: `You are a Writer. Write a single scene from the brief you receive.

Rules:
- Write exactly 2-3 sentences. Each sentence max 10 words.
- Be vivid but concise.

When finished, send the FULL scene text to your leader in a single message. Do NOT send a summary or completion notice — send the actual written text.`,
		},
	},

	Environments: []EnvironmentDef{
		{Name: "eic-env", Key: "STORY_ROLE", Value: "editor-in-chief"},
		{Name: "se-env", Key: "STORY_ROLE", Value: "section-editor"},
		{Name: "writer-env", Key: "STORY_ROLE", Value: "writer"},
	},

	GitRepos: []GitRepoDef{
		{Name: "story-repo", URL: "https://github.com/jakeraft/clier_hello.git"},
	},

	CliProfiles: []CliProfileDef{
		{Name: "claude-sonnet", PresetKey: "claude-sonnet"},
		{Name: "codex", PresetKey: "codex-5.4"},
	},

	Members: []MemberDef{
		{Name: "chief", CliProfileName: "claude-sonnet", SystemPromptNames: []string{"editor-in-chief"}, EnvNames: []string{"eic-env"}, GitRepoName: "story-repo"},
		{Name: "se-1", CliProfileName: "claude-sonnet", SystemPromptNames: []string{"section-editor"}, EnvNames: []string{"se-env"}, GitRepoName: "story-repo"},
		{Name: "se-2", CliProfileName: "claude-sonnet", SystemPromptNames: []string{"section-editor"}, EnvNames: []string{"se-env"}, GitRepoName: "story-repo"},
		{Name: "writer-1", CliProfileName: "claude-sonnet", SystemPromptNames: []string{"writer"}, EnvNames: []string{"writer-env"}, GitRepoName: "story-repo"},
		{Name: "writer-2", CliProfileName: "claude-sonnet", SystemPromptNames: []string{"writer"}, EnvNames: []string{"writer-env"}, GitRepoName: "story-repo"},
		{Name: "writer-3", CliProfileName: "codex", SystemPromptNames: []string{"writer"}, EnvNames: []string{"writer-env"}},
		{Name: "writer-4", CliProfileName: "codex", SystemPromptNames: []string{"writer"}, EnvNames: []string{"writer-env"}},
	},

	Team: TeamDef{
		Name:           "story-team",
		RootMemberName: "chief",
	},

	Relations: []RelationDef{
		{From: "chief", To: "se-1", Type: domain.RelationLeader},
		{From: "chief", To: "se-2", Type: domain.RelationLeader},
		{From: "se-1", To: "writer-1", Type: domain.RelationLeader},
		{From: "se-1", To: "writer-2", Type: domain.RelationLeader},
		{From: "se-2", To: "writer-3", Type: domain.RelationLeader},
		{From: "se-2", To: "writer-4", Type: domain.RelationLeader},
	},
}
