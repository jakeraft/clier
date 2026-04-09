package cmd

import (
	"errors"
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
		Short:   "Compose and run agent teams",
		GroupID: rootGroupServer,
		Long: `Compose and run agent teams.

Use list, view, create, edit, delete, and fork to manage your
team definitions. Use download and run to bring them to life locally.

Workflow:
  clier team create          Define a new team
  clier team download <name> Pull it to your machine
  clier team run             Start all agents in tmux`,
	}
	cmd.AddGroup(
		&cobra.Group{ID: subGroupServer, Title: "Define"},
		&cobra.Group{ID: subGroupRuntime, Title: "Run"},
	)
	cmd.AddCommand(newTeamListCmd())
	cmd.AddCommand(newTeamViewCmd())
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamEditCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	cmd.AddCommand(newTeamForkCmd())
	cmd.AddCommand(newTeamDownloadCmd())
	cmd.AddCommand(newTeamRunCmd())
	return cmd
}

func newTeamListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list [owner]",
		Short:   "List your teams",
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
		Short:   "Show team details",
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
		Short:   "Create a new team",
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
				return errors.New("--root-index must be set to a non-negative team_members index")
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
		Short:   "Update a team",
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
					return errors.New("--root-index is required when replacing --member because team membership is index-based")
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
		Short:   "Delete a team",
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
		Short:   "Copy a public team to your namespace",
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

func newTeamDownloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "download <[owner/]name>",
		Aliases: []string{"clone", "workspace"},
		Short:   "Download a team to a local directory",
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
				"status": "downloaded",
				"team":   name,
				"dir":    teamBase,
			})
		},
	}
}

func newTeamRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "run",
		Short:   "Start all agents in tmux",
		GroupID: subGroupRuntime,
		Long: `Start all team agents in a tmux session.

Run this from the workspace directory created by ` + "`team download`" + `.
The current directory must contain ` + "`.clier/workspace.json`" + `.
Each agent gets its own tmux window within a single session.

To refresh a workspace, download it again.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			teamBase, meta, err := requireCurrentCloneRootKind(resourceKindTeam, "`clier team run`")
			if err != nil {
				return err
			}
			if err := validateDownloadedWorkspace(teamBase, meta); err != nil {
				return err
			}
			team := meta.Workspace.Team

			runID, err := newRunID()
			if err != nil {
				return err
			}
			runName := apprun.SessionName(team.Name, runID)

			var terminalPlans []apprun.MemberTerminal

			for i, tm := range team.Members {
				memberBase := filepath.Join(teamBase, tm.Name)
				repoPath := memberBase

				envVars := buildMemberEnv(runID, tm.TeamMemberID, &team.ID, tm.Name)
				fullCommand := buildFullCommand(envVars, tm.Command, repoPath)

				terminalPlans = append(terminalPlans, apprun.MemberTerminal{
					TeamMemberID: tm.TeamMemberID,
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
