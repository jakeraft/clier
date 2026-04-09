package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/auth"
	"github.com/jakeraft/clier/internal/config"
	"github.com/spf13/cobra"
)

const (
	rootGroupServer    = "server"
	rootGroupRuntime   = "runtime"
	rootGroupDiscovery = "discovery"
	rootGroupLocal     = "local"

	subGroupServer  = "server"
	subGroupRuntime = "runtime"
)

// newAPIClient creates an API client.
// Token is loaded from credentials if available, empty otherwise.
func newAPIClient() *api.Client {
	cfg := currentConfig()

	token := ""
	creds, err := auth.Load(cfg.CredentialsPath)
	if err == nil {
		token = creds.Token
	}
	return api.NewClient(cfg.ServerURL, token)
}

func loadRawConfig() (*config.File, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	return cfg, nil
}

func loadConfig() (*config.File, error) {
	raw, err := loadRawConfig()
	if err != nil {
		return nil, err
	}
	return config.Resolve(raw)
}

func currentConfig() *config.File {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load clier config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func configPath() string {
	path, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve clier config path: %v\n", err)
		os.Exit(1)
	}
	return path
}

func newTerminal() *terminal.TmuxTerminal {
	return terminal.NewTmuxTerminal()
}

// requireLogin loads credentials and returns login.
// Exits with error if not logged in.
func requireLogin() string {
	creds, err := auth.Load(currentConfig().CredentialsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: not logged in. Run 'clier auth login' first.")
		os.Exit(1)
	}
	return creds.Login
}

var rootCmd = &cobra.Command{
	Use:   "clier",
	Short: "Orchestrate AI coding agent teams in isolated local clones",
	Long: `clier manages reusable agent resources on clier-server and materializes
them into runnable local clones.

Command groups:
  Resource commands
    These talk to clier-server and manage shared resources such as
    members, teams, claude-md files, claude-settings files, skills,
    auth, and discovery.

  Clone and run commands
    These materialize local member clones or team clones and run local
    member agents in tmux. ` + "`member clone`" + `, ` + "`member run`" + `,
    ` + "`team clone`" + `, ` + "`team run`" + `, and ` + "`run ...`" + `
    are all local commands.

Common starting points:
  Work with a single member
    ` + "`clier member --help`" + `

  Work with a team and its member agents
    ` + "`clier team --help`" + `

  Operate an existing local run
    ` + "`clier run --help`" + `

  Learn the workflow end to end
    ` + "`clier tutorial`" + `

Clone model:
  A clone root is the directory that directly owns ` + "`.clier/clone.json`" + `.
  Use ` + "`member run`" + ` and ` + "`team run`" + ` from that clone root.
  Use ` + "`run ...`" + ` from anywhere inside that clone.

  Clones are one-way materializations from clier-server resources.
  Local file edits do not sync back to the server. To change server state,
  use explicit resource commands such as ` + "`member edit`" + `, ` + "`team edit`" + `,
  ` + "`claudemd edit`" + `, ` + "`claudesettings edit`" + `, and ` + "`skill edit`" + `.
  To refresh a clone, remove it and clone again.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() {
	configureCommandGroups()
	cmd := rootCmd
	if isAgentMode() {
		cmd = newAgentRootCmd(isTeamAgent())
	} else {
		filterUserCommands()
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configureCommandGroups() {
	rootCmd.AddGroup(
		&cobra.Group{ID: rootGroupServer, Title: "Server-Backed Resource Commands"},
		&cobra.Group{ID: rootGroupRuntime, Title: "Local Runtime Commands"},
		&cobra.Group{ID: rootGroupDiscovery, Title: "Discovery Commands"},
		&cobra.Group{ID: rootGroupLocal, Title: "Local Configuration Commands"},
	)
}

// filterUserCommands removes agent-only subcommands from "run" in user context.
func filterUserCommands() {
	// Coupled to: newRunNoteCmd
	hidden := map[string]bool{"note": true}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "run" {
			var keep []*cobra.Command
			for _, sub := range cmd.Commands() {
				if !hidden[sub.Name()] {
					keep = append(keep, sub)
				}
			}
			cmd.ResetCommands()
			for _, sub := range keep {
				cmd.AddCommand(sub)
			}
		}
	}
}

// parseOwnerName splits "owner/name" into owner and name.
// If no slash is present, requireLogin() is used as owner.
func parseOwnerName(s string) (owner, name string) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return requireLogin(), s
}

func newAgentRootCmd(teamScoped bool) *cobra.Command {
	root := &cobra.Command{
		Use:   "clier",
		Short: "Commands for this run",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	if teamScoped {
		root.Long = "Use `clier run tell` to message another team member.\nUse `clier run note` to record a work log entry."
	} else {
		root.Long = "Use `clier run note` to record a work log entry."
	}
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.SetHelpTemplate(`{{with (or .Long .Short)}}{{.}}
{{end}}{{if or .Runnable .HasSubCommands}}{{if .HasSubCommands}}
Usage:
  {{.UseLine}}

Available Commands:{{range .Commands}}{{if (and .IsAvailableCommand (not .Hidden))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}{{end}}
`)
	root.AddCommand(newAgentRunCmd(teamScoped))
	return root
}

func newAgentRunCmd(teamScoped bool) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Commands for this run",
	}
	if teamScoped {
		run.Long = "Use `tell` to message another team member.\nUse `note` to record a work log entry."
		run.AddCommand(newRunTellCmd())
	} else {
		run.Long = "Use `note` to record a work log entry."
	}
	run.AddCommand(newRunNoteCmd())
	return run
}
