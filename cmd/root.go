package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	adaptergit "github.com/jakeraft/clier/internal/adapter/git"
	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/auth"
	"github.com/jakeraft/clier/internal/config"
	"github.com/spf13/cobra"
)

const (
	rootGroupResources = "resources"
	rootGroupRuntime   = "runtime"
	rootGroupWorkspace = "workspace"
	rootGroupSettings  = "settings"
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

func newFileMaterializer() *filesystem.LocalFS {
	return filesystem.New()
}

func newGitRepo() *adaptergit.ExecGit {
	return adaptergit.New()
}

func newTerminal() *terminal.TmuxTerminal {
	return terminal.NewTmuxTerminal()
}

// SetVersion sets the CLI version string shown by --version.
func SetVersion(v string) {
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "clier",
	Short: "Harness AI agent teams",
	Long: `clier is a harness for AI coding agent teams.

Define agents, compose them into teams, and run them locally in tmux.
Each agent gets its own terminal, git repo, and system prompt.
You watch, steer, and intervene in real time.

Teams and individual members can both be cloned and run.
Cloning a member creates a runnable 1-member workspace.

Get started:
  clier tutorial               Walk through an example team
  clier list --kind team       Browse what others have built

Core workflow:
  clier create member            Define an agent
  clier create team              Compose agents into a team
  clier fork <owner/name>        Fork a resource to customize it
  clier clone <owner/name>       Download a local working copy
  clier run start                Launch agents in tmux
  clier run tell --to <name>     Send instructions to an agent
  clier run attach <run-id>      Watch agents in real time
  clier open dashboard           Open the dashboard in a browser`,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// errSubcommandRequired is returned by parent commands called without a subcommand.
var errSubcommandRequired = errors.New("subcommand required")

// subcommandRequired prints help and returns errSubcommandRequired (exit 1).
func subcommandRequired(cmd *cobra.Command, _ []string) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	_ = cmd.Help()
	return errSubcommandRequired
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
		if !errors.Is(err, errSubcommandRequired) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

func configureCommandGroups() {
	rootCmd.AddGroup(
		&cobra.Group{ID: rootGroupResources, Title: "Resources"},
		&cobra.Group{ID: rootGroupRuntime, Title: "Runtime"},
		&cobra.Group{ID: rootGroupWorkspace, Title: "Workspace"},
		&cobra.Group{ID: rootGroupSettings, Title: "Settings"},
	)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true, GroupID: rootGroupSettings})
}

// filterUserCommands removes agent-only subcommands from "run" in user context.
func filterUserCommands() {
	agentOnly := map[string]bool{cmdNameNote: true}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == cmdNameRun {
			var keep []*cobra.Command
			for _, sub := range cmd.Commands() {
				if !agentOnly[sub.Name()] {
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

// currentLogin returns the logged-in user's login, or empty string if not logged in.
func currentLogin() string {
	creds, err := auth.Load(currentConfig().CredentialsPath)
	if err != nil {
		return ""
	}
	return creds.Login
}

// resolveOwner returns the explicit owner if set, otherwise falls back to logged-in user.
func resolveOwner(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	login := currentLogin()
	if login == "" {
		return "", errors.New("specify --owner or run 'clier auth login'")
	}
	return login, nil
}

// parseOwnerName splits "owner/name" into owner and name.
func parseOwnerName(s string) (owner, name string, err error) {
	parts := strings.SplitN(strings.TrimSpace(s), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid resource %q: want <owner/name>", s)
	}
	return parts[0], parts[1], nil
}

func newAgentRootCmd(teamScoped bool) *cobra.Command {
	root := &cobra.Command{
		Use:   "clier",
		Short: "Commands for this run",
		SilenceErrors: true,
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
