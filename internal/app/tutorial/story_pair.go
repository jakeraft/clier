package tutorial

import "github.com/jakeraft/clier/internal/domain"

func init() {
	Register(StoryPair)
}

var teamProtocolPrompt = `# Team Protocol

You are a member of a team managed by clier.
Before taking any action, run the following command to discover your role and teammates:

` + "```bash\nclier sprint whoami\n```" + `

This returns your member ID, name, team name, and relations (leaders, workers, peers).

## Communication

Send a message to a teammate:

` + "```bash\nclier message send --to <member-id> \"<message>\"\n```" + `

Replies arrive directly in your terminal input. Do not poll or call any receive command.

## Behavioral Rules

- **If you have a leader** — report your results to them when done.
- **If you have workers** — delegate sub-tasks to them and wait for all responses before wrapping up.
- **If you have peers** — coordinate with them when tasks overlap.
- **If you have no leader** — you were started by a human user. Report final results with:

` + "```bash\nclier message send --to " + domain.UserMemberID + " \"<message>\"\n```\n"

var StoryPair = &Scenario{
	Name:        "story-pair",
	Description: "Simple 2-member team: editor delegates to one writer",

	SystemPrompts: []SystemPromptDef{
		{
			Name:   "Team Protocol",
			Prompt: teamProtocolPrompt,
		},
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
		{Name: "editor", CliProfileName: "pair-sonnet", SystemPromptNames: []string{"Team Protocol", "pair-editor"}, GitRepoName: "pair-repo"},
		{Name: "writer", CliProfileName: "pair-sonnet", SystemPromptNames: []string{"Team Protocol", "pair-writer"}, GitRepoName: "pair-repo"},
	},

	Team: TeamDef{
		Name:           "story-pair",
		RootMemberName: "editor",
	},

	Relations: []RelationDef{
		{From: "editor", To: "writer", Type: domain.RelationLeader},
	},
}
