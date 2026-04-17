package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/jakeraft/clier/cmd/middleware"
	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/app"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	adaptergit "github.com/jakeraft/clier/internal/adapter/git"
	"github.com/jakeraft/clier/internal/adapter/terminal"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/auth"
	"github.com/jakeraft/clier/internal/config"
	"github.com/jakeraft/clier/internal/domain"
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
	Use:           "clier",
	Short:         "Harness multi-agent teams with a native CLI",
	SilenceUsage:  true,
	Long: `clier is a harness for AI coding agent teams.

Define agents, compose them into teams, and run them locally in tmux.
Each agent gets its own terminal, git repo, and system prompt.
You watch, steer, and intervene in real time.

Get started:
  clier tutorial               Walk through the hello-claude team
  clier list --kind team       Browse what others have built

Try a team:
  clier clone <owner/name>                  Download a working copy
  clier run start <owner/name>              Launch agents in tmux
  clier run tell --run <id> --to <agent>    Send instructions to an agent
  clier run attach <run-id>                 Watch agents in real time
  clier remove <owner/name>                 Delete a working copy and its runs

Refine and share:
  clier create team                         Define a new team
  clier fork <owner/name>                   Fork an existing resource to your namespace
  clier status <owner/name>                 Show local modifications
  clier push <owner/name>                   Publish your refinements
  clier pull <owner/name>                   Sync the latest from the registry
  clier open dashboard                      Open the dashboard in a browser

For agent consumers:
  clier output is shaped for agents to parse. Some commands include
  a "hint" field describing a next step. When the hint instructs a
  user-only action (e.g., "clier run attach" needs a real terminal),
  copy the relevant command to the user's clipboard (pbcopy / xclip
  / clip.exe) and ask the user to run it. The agent drives the
  workflow; the user is the keyboard for human-only steps.

Output conventions:
  - JSON object on stdout, snake_case fields across all commands.
  - Empty collections render as [] (never null).
  - Server timestamps are RFC3339 UTC (Z suffix); runtime timestamps
    from the host carry the local offset.`,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// errSubcommandRequired is returned by parent commands called without a subcommand.
var errSubcommandRequired = errors.New("subcommand required")

// errSilent signals exit code 1 without any additional "Error:" prefix.
// Use when the command has already printed a user-facing message and just
// needs the process to exit non-zero.
var errSilent = errors.New("silent exit")

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

	middleware.Apply(cmd, middleware.Chain(
		middleware.Recover,
		middleware.Translate,
	))

	if err := cmd.Execute(); err != nil {
		if errors.Is(err, errSubcommandRequired) || errors.Is(err, errSilent) {
			os.Exit(1)
		}
		// Args validators run outside RunE, so flag/positional failures
		// bypass the middleware chain. Translate once more here so the
		// presenter always sees a domain.Fault.
		present.Emit(os.Stderr, app.Translate(err))
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
	rootCmd.SetHelpCommandGroupID(rootGroupSettings)
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
		return "", &domain.Fault{Kind: domain.KindOwnerRequired}
	}
	return login, nil
}

func splitResourceID(id string) (owner, name string, err error) {
	return appworkspace.SplitResourceID(id)
}

func newAgentRootCmd(teamScoped bool) *cobra.Command {
	root := &cobra.Command{
		Use:           "clier",
		Short:         "Commands for this run",
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	if teamScoped {
		root.Long = "Use `clier run tell` to message another agent.\nUse `clier run note` to record a work log entry."
	} else {
		root.Long = "Use `clier run note` to record a work log entry."
	}
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
		run.Long = "Use `tell` to message another agent.\nUse `note` to record a work log entry."
		run.AddCommand(newRunTellCmd())
	} else {
		run.Long = "Use `note` to record a work log entry."
	}
	run.AddCommand(newRunNoteCmd())
	return run
}
