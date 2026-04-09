package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
	appclone "github.com/jakeraft/clier/internal/app/clone"
	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTeamCmd())
}

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "team",
		Short:   "Manage teams",
		GroupID: rootGroupServer,
		Long: `Manage team resources and team clones.

Server-backed subcommands:
  list, view, create, edit, delete, fork

Local runtime subcommands:
  clone, run

Use ` + "`team clone`" + ` to materialize a local team clone under
` + "`./<owner>/<name>`" + `. Use ` + "`team run`" + ` from that clone root
to launch a tmux session with one window per team member.

` + "`team clone`" + ` is one-way: it writes local member worktrees and
team protocol files, but does not sync local file edits back to
clier-server. Update server resources with explicit resource commands,
then remove and re-clone when you want a fresh local copy.`,
	}
	cmd.AddGroup(
		&cobra.Group{ID: subGroupServer, Title: "Server-Backed Team Commands"},
		&cobra.Group{ID: subGroupRuntime, Title: "Local Runtime Commands"},
	)
	cmd.AddCommand(newTeamListCmd())
	cmd.AddCommand(newTeamViewCmd())
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamEditCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	cmd.AddCommand(newTeamForkCmd())
	cmd.AddCommand(newTeamCloneCmd())
	cmd.AddCommand(newTeamRunCmd())
	return cmd
}

func newTeamListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [owner]",
		Short:   "List teams from clier-server",
		Long:    "List your teams, or another user's teams if [owner] is given.",
		GroupID: subGroupServer,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			var owner string
			if len(args) == 1 {
				owner = args[0]
			} else {
				owner = requireLogin()
			}
			teams, err := client.ListTeams(owner)
			if err != nil {
				return err
			}
			return printJSON(teams)
		},
	}
}

func newTeamViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <[owner/]name>",
		Short:   "View a team from clier-server",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			team, err := client.GetTeam(owner, name)
			if err != nil {
				return err
			}
			return printJSON(team)
		},
	}
}

func newTeamCreateCmd() *cobra.Command {
	var name string
	var teamMembers, relations []string
	rootIndex := -1

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a team on clier-server",
		GroupID: subGroupServer,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			members, err := parseTeamMemberSpecs(teamMembers)
			if err != nil {
				return err
			}
			parsedRelations, err := parseTeamRelationSpecs(relations)
			if err != nil {
				return err
			}
			if rootIndex < 0 {
				return fmt.Errorf("--root-index must be set to a non-negative team_members index")
			}
			body := api.TeamMutationRequest{
				Name:        name,
				TeamMembers: members,
				Relations:   parsedRelations,
				RootIndex:   &rootIndex,
			}
			resp, err := client.CreateTeam(owner, body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Team member as <member-id>:<name>; repeat for each member")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Relation as <from-index>:<to-index> using zero-based --member indices; repeat for each edge")
	cmd.Flags().IntVar(&rootIndex, "root-index", -1, "Root member index in the zero-based --member list")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("member")
	_ = cmd.MarkFlagRequired("root-index")
	return cmd
}

func newTeamEditCmd() *cobra.Command {
	var name string
	var teamMembers, relations []string
	rootIndex := -1

	cmd := &cobra.Command{
		Use:     "edit <name>",
		Short:   "Edit a team on clier-server",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			current, err := client.GetTeam(owner, args[0])
			if err != nil {
				return err
			}
			body, err := teamMutationRequestFromResponse(current)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				body.Name = name
			}
			if cmd.Flags().Changed("member") {
				body.TeamMembers, err = parseTeamMemberSpecs(teamMembers)
				if err != nil {
					return err
				}
				if !cmd.Flags().Changed("relation") {
					body.Relations = nil
				}
				if !cmd.Flags().Changed("root-index") {
					return fmt.Errorf("--root-index is required when replacing --member because team membership is index-based")
				}
			}
			if cmd.Flags().Changed("relation") {
				body.Relations, err = parseTeamRelationSpecs(relations)
				if err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("root-index") {
				if rootIndex < 0 {
					body.RootIndex = nil
				} else {
					body.RootIndex = &rootIndex
				}
			}
			resp, err := client.UpdateTeam(owner, args[0], body)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New team name")
	cmd.Flags().StringSliceVar(&teamMembers, "member", nil, "Replace team members with <member-id>:<name>; repeat for each member")
	cmd.Flags().StringSliceVar(&relations, "relation", nil, "Replace relations with <from-index>:<to-index> using zero-based member indices; repeat for each edge")
	cmd.Flags().IntVar(&rootIndex, "root-index", -1, "Replace root member index; use -1 to clear")
	return cmd
}

func newTeamDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a team from clier-server",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner := requireLogin()
			if err := client.DeleteTeam(owner, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}

func newTeamForkCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "fork <owner/name>",
		Short:   "Fork a team on clier-server to your namespace",
		GroupID: subGroupServer,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			_ = requireLogin()
			owner, name := parseOwnerName(args[0])
			resp, err := client.ForkTeam(owner, name)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}

func newTeamCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "clone <[owner/]name>",
		Aliases: []string{"workspace"},
		Short:   "Create a local team clone under ./<owner>/<name>",
		GroupID: subGroupRuntime,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			writer := appclone.NewWriter(client, owner)
			teamBase, err := resolveCloneCreateBase(cloneTarget{
				Kind:  resourceKindTeam,
				Owner: owner,
				Name:  name,
			})
			if err != nil {
				return err
			}

			if err := writer.PrepareTeam(teamBase, name); err != nil {
				return err
			}
			meta, err := buildTeamCloneMetadata(client, owner, name)
			if err != nil {
				return err
			}
			if err := appclone.SaveCloneMetadata(teamBase, meta); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "cloned",
				"team":   name,
				"dir":    teamBase,
			})
		},
	}
}

func newTeamRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "run <[owner/]name>",
		Short:   "Launch a local team run from the current clone root",
		GroupID: subGroupRuntime,
		Long: `Launch a team run from the current clone root.
This command is local runtime, not a clier-server run API call.

The current directory must be the team clone root that directly owns
` + "`.clier/clone.json`" + ` for the requested team. Run ` + "`team clone`" + `
first, then ` + "`cd`" + ` into that clone root before starting a run.
Each member gets its own tmux window within a single session.

The clone is a one-way local worktree. To refresh it from server
resources, remove the clone and create it again.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			owner, name := parseOwnerName(args[0])
			_ = requireLogin()

			team, err := client.GetTeam(owner, name)
			if err != nil {
				return fmt.Errorf("get team: %w", err)
			}
			teamBase, _, err := requireCurrentCloneRoot(cloneTarget{
				Kind:  resourceKindTeam,
				Owner: owner,
				Name:  team.Name,
			}, "`clier team run`")
			if err != nil {
				return err
			}

			writer := appclone.NewWriter(client, owner)
			needsPrepare := false
			memberResponses := make(map[string]*api.MemberResponse, len(team.TeamMembers))
			for _, tm := range team.TeamMembers {
				member, err := client.GetMember(tm.Member.Owner, tm.Member.Name)
				if err != nil {
					return fmt.Errorf("get member %s: %w", tm.Name, err)
				}
				memberResponses[tm.Name] = member
				memberBase := filepath.Join(teamBase, tm.Name)
				prepared, err := appclone.IsPreparedRoot(member.GitRepoURL, memberBase)
				if err != nil {
					return err
				}
				if !prepared {
					needsPrepare = true
				}
			}
			if needsPrepare {
				if err := writer.PrepareTeam(teamBase, name); err != nil {
					return fmt.Errorf("prepare team clone: %w", err)
				}
				meta, err := buildTeamCloneMetadata(client, owner, name)
				if err != nil {
					return err
				}
				if err := appclone.SaveCloneMetadata(teamBase, meta); err != nil {
					return err
				}
			}

			runID, err := newRunID()
			if err != nil {
				return err
			}
			runName := apprun.SessionName(team.Name, runID)

			var terminalPlans []apprun.MemberTerminal

			for i, tm := range team.TeamMembers {
				member := memberResponses[tm.Name]
				memberBase := filepath.Join(teamBase, tm.Name)
				repoPath := memberBase

				envVars := buildMemberEnv(runID, tm.ID, &team.ID, tm.Name)
				fullCommand := buildFullCommand(envVars, member.Command, repoPath)

				terminalPlans = append(terminalPlans, apprun.MemberTerminal{
					TeamMemberID: tm.ID,
					Name:         tm.Name,
					Window:       i,
					Memberspace:  memberBase,
					Cwd:          repoPath,
					Command:      fullCommand,
				})
			}

			runner := apprun.NewRunner(newTerminal())
			plan, err := runner.Run(teamBase, runID, runName, terminalPlans)
			if err != nil {
				return err
			}

			return printJSON(map[string]any{
				"run_id":  runID,
				"session": plan.Session,
			})
		},
	}
}
