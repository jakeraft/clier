package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTeamCmd())
}

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Browse and manage teams on the server",
		Args:  cobra.ArbitraryArgs,
		RunE:  helpOrUnknown,
	}
	cmd.AddCommand(
		newTeamListCmd(),
		newTeamGetCmd(),
		newTeamCreateCmd(),
		newTeamUpdateCmd(),
		newTeamDeleteCmd(),
		newTeamStarCmd(),
		newTeamUnstarCmd(),
	)
	return cmd
}

func newTeamListCmd() *cobra.Command {
	var (
		namespace string
		agentType string
		sort      string
		q         string
		pageSize  int
		pageToken string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List teams (cursor pagination, default sort=stars_desc)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := newAPIClient()
			if err != nil {
				return err
			}
			// page-size: cmd.Flags().Changed lets the CLI tell apart "user
			// passed --page-size N" from "user omitted the flag". omit ⇒
			// nil ⇒ server default. Explicit 0 / negative ⇒ ptr ⇒ server
			// 422 (don't silently swallow user input on the way out).
			var pageSizePtr *int
			if cmd.Flags().Changed("page-size") {
				pageSizePtr = &pageSize
			}
			res, err := client.ListTeams(api.ListTeamsQuery{
				Namespace: namespace,
				AgentType: agentType,
				Sort:      sort,
				Q:         q,
				PageSize:  pageSizePtr,
				PageToken: pageToken,
			})
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), res)
		},
	}
	cmd.Flags().StringVar(&namespace, "namespace", "", "Filter by owning namespace")
	cmd.Flags().StringVar(&agentType, "agent-type", "", "Filter by agent type (claude|codex)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort enum (stars_desc|stars_asc|updated_desc|updated_asc)")
	cmd.Flags().StringVar(&q, "q", "", "Substring match on name or description")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Max items per page (server default 20, max 100)")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Cursor token from a previous response's meta.next_cursor")
	return cmd
}

func newTeamGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <namespace/name>",
		Short: "Show one team by natural key",
		Args:  requireOneArg("<namespace/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, name, err := splitTeamID(args[0])
			if err != nil {
				return err
			}
			client, _, err := newAPIClient()
			if err != nil {
				return err
			}
			team, err := client.GetTeam(ns, name)
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), team)
		},
	}
}

func newTeamCreateCmd() *cobra.Command {
	var (
		description string
		agentType   string
		command     string
		gitRepoURL  string
		gitSubpath  string
		subteamRefs []string
	)
	cmd := &cobra.Command{
		Use:   "create <namespace/name>",
		Short: "Register a new team in your namespace",
		Long: `Register a new team. The actor must own the target namespace
(ADR-0005 §3.1) — typically your own GitHub login.

The server-default protocol template is auto-injected at create time.
Edit the rendered protocol later via 'clier team update --protocol-file'
(future) or the dashboard.`,
		Args: requireOneArg("<namespace/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, name, err := splitTeamID(args[0])
			if err != nil {
				return err
			}
			subs, err := parseSubteamRefs(subteamRefs)
			if err != nil {
				return err
			}
			client, _, err := newAPIClient()
			if err != nil {
				return err
			}
			team, err := client.CreateTeam(api.CreateTeamRequest{
				Namespace:   ns,
				Name:        name,
				Description: description,
				AgentType:   agentType,
				Command:     command,
				GitRepoURL:  gitRepoURL,
				GitSubpath:  gitSubpath,
				Subteams:    subs,
			})
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), team)
		},
	}
	// `(required)` / `(optional)` suffixes are visible in `--help` output —
	// the QA agent (qa-20260504192453, help.required-flags-not-marked) noted
	// that cobra's MarkFlagRequired does not annotate the help text, so an
	// agent reading `team create --help` cannot tell which flags are
	// mandatory without provoking the failure. The marker is the contract.
	cmd.Flags().StringVar(&agentType, "agent-type", "", "Agent type — claude | codex (required)")
	cmd.Flags().StringVar(&command, "command", "", "Vendor binary + flags, verbatim send-keys (required)")
	cmd.Flags().StringVar(&gitRepoURL, "git-repo-url", "", "GitHub HTTPS URL — https://github.com/owner/repo (required)")
	cmd.Flags().StringVar(&description, "description", "", "Team description (optional)")
	cmd.Flags().StringVar(&gitSubpath, "git-subpath", "", "Repo-relative cwd offset, empty = repo root (optional)")
	cmd.Flags().StringSliceVar(&subteamRefs, "subteam", nil, "Direct child team — namespace/name, repeatable (optional)")
	_ = cmd.MarkFlagRequired("agent-type")
	_ = cmd.MarkFlagRequired("command")
	_ = cmd.MarkFlagRequired("git-repo-url")
	return cmd
}

func newTeamUpdateCmd() *cobra.Command {
	var (
		description    *string
		command        *string
		gitRepoURL     *string
		gitSubpath     *string
		subteamRefs    []string
		subteamsActive bool
		patchJSON      string
	)
	descFlag := newOptStringFlag(&description)
	cmdFlag := newOptStringFlag(&command)
	repoFlag := newOptStringFlag(&gitRepoURL)
	pathFlag := newOptStringFlag(&gitSubpath)
	cmd := &cobra.Command{
		Use:   "update <namespace/name>",
		Short: "Patch a team (RFC 7396 JSON Merge Patch)",
		Long: `Update mutable team fields via JSON Merge Patch (RFC 7396).
Only the flags you pass are sent — omitted fields stay unchanged.
Immutable fields (namespace, name, agent_type) cannot be patched.

For complex updates use --patch-json with a literal merge-patch body.`,
		Args: requireOneArg("<namespace/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, name, err := splitTeamID(args[0])
			if err != nil {
				return err
			}
			subteamsActive = cmd.Flags().Changed("subteam")
			patch, err := buildTeamPatch(description, command, gitRepoURL, gitSubpath, subteamRefs, subteamsActive, patchJSON)
			if err != nil {
				return err
			}
			if len(patch) == 0 {
				return fmt.Errorf("no fields to update — pass at least one --description / --command / --git-repo-url / --git-subpath / --subteam / --patch-json")
			}
			client, _, err := newAPIClient()
			if err != nil {
				return err
			}
			team, err := client.UpdateTeam(ns, name, patch)
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), team)
		},
	}
	cmd.Flags().Var(descFlag, "description", "New description")
	cmd.Flags().Var(cmdFlag, "command", "New vendor command line")
	cmd.Flags().Var(repoFlag, "git-repo-url", "New GitHub HTTPS URL")
	cmd.Flags().Var(pathFlag, "git-subpath", "New repo-relative subpath (empty string = repo root)")
	cmd.Flags().StringSliceVar(&subteamRefs, "subteam", nil, "Replace subteam list with these (namespace/name); repeatable, pass with no values to clear")
	cmd.Flags().StringVar(&patchJSON, "patch-json", "", "Raw JSON Merge Patch body (overrides per-field flags)")
	return cmd
}

func newTeamDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <namespace/name>",
		Short: "Delete a team (204 on success)",
		Args:  requireOneArg("<namespace/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, name, err := splitTeamID(args[0])
			if err != nil {
				return err
			}
			client, _, err := newAPIClient()
			if err != nil {
				return err
			}
			if err := client.DeleteTeam(ns, name); err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"namespace": ns,
				"name":      name,
				"deleted":   true,
			})
		},
	}
}

func newTeamStarCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "star <namespace/name>",
		Short: "Star a team (idempotent)",
		Args:  requireOneArg("<namespace/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, name, err := splitTeamID(args[0])
			if err != nil {
				return err
			}
			client, _, err := newAPIClient()
			if err != nil {
				return err
			}
			if err := client.StarTeam(ns, name); err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"namespace": ns,
				"name":      name,
				"starred":   true,
			})
		},
	}
}

func newTeamUnstarCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unstar <namespace/name>",
		Short: "Remove the star (idempotent)",
		Args:  requireOneArg("<namespace/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, name, err := splitTeamID(args[0])
			if err != nil {
				return err
			}
			client, _, err := newAPIClient()
			if err != nil {
				return err
			}
			if err := client.UnstarTeam(ns, name); err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"namespace": ns,
				"name":      name,
				"starred":   false,
			})
		},
	}
}

// parseSubteamRefs converts ["ns/name", "ns2/name2"] into TeamKey slice.
// Empty input returns nil so the API client omits the field entirely.
func parseSubteamRefs(raw []string) ([]api.TeamKey, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]api.TeamKey, 0, len(raw))
	for _, ref := range raw {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		ns, name, err := splitTeamID(ref)
		if err != nil {
			return nil, fmt.Errorf("subteam %q: %w", ref, err)
		}
		out = append(out, api.TeamKey{Namespace: ns, Name: name})
	}
	return out, nil
}

// buildTeamPatch composes the merge-patch body. --patch-json wins entirely
// when set (escape hatch); otherwise per-field flags are merged in. Each
// optional pointer is only included when the user actually passed the flag
// — distinguishing "unchanged" from "set to empty string" requires the
// pointer wrap (cobra's StringVar conflates both).
func buildTeamPatch(description, command, gitRepoURL, gitSubpath *string,
	subteamRefs []string, subteamsActive bool, patchJSON string) (map[string]any, error) {
	if patchJSON != "" {
		var raw map[string]any
		if err := json.Unmarshal([]byte(patchJSON), &raw); err != nil {
			return nil, fmt.Errorf("--patch-json: %w", err)
		}
		return raw, nil
	}
	patch := map[string]any{}
	if description != nil {
		patch["description"] = *description
	}
	if command != nil {
		patch["command"] = *command
	}
	if gitRepoURL != nil {
		patch["git_repo_url"] = *gitRepoURL
	}
	if gitSubpath != nil {
		patch["git_subpath"] = *gitSubpath
	}
	if subteamsActive {
		subs, err := parseSubteamRefs(subteamRefs)
		if err != nil {
			return nil, err
		}
		// Server expects []TeamKey; nil → empty slice so the patch
		// signals "clear all subteams" rather than "leave unchanged".
		if subs == nil {
			subs = []api.TeamKey{}
		}
		patch["subteams"] = subs
	}
	return patch, nil
}

// optStringFlag is a pflag.Value that captures whether the user passed the
// flag at all. nil pointer = absent; non-nil = present (even if empty).
// Lets the patch builder distinguish "leave unchanged" from "set to empty
// string" — cobra's plain StringVar conflates the two.
type optStringFlag struct {
	target **string
}

func newOptStringFlag(target **string) *optStringFlag {
	return &optStringFlag{target: target}
}

func (f *optStringFlag) String() string {
	if f.target == nil || *f.target == nil {
		return ""
	}
	return **f.target
}

func (f *optStringFlag) Set(v string) error {
	*f.target = &v
	return nil
}

func (f *optStringFlag) Type() string { return "string" }
