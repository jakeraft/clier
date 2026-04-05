package cmd

import (
	"errors"
	"os"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/adapter/workspace"
	"github.com/jakeraft/clier/internal/app/task"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTaskCmd())
}

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}
	cmd.AddCommand(newTaskStartCmd())
	cmd.AddCommand(newTaskStopCmd())
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskTellCmd())
	cmd.AddCommand(newTaskNoteCmd())
	cmd.AddCommand(newTaskNotesCmd())
	cmd.AddCommand(newTaskAttachCmd())
	return cmd
}

func newTaskStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "start <team-id>",
		Short:       "Start a task",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			t, err := store.GetTeam(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := task.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			tk, err := svc.Start(cmd.Context(), t, cfg.Auth)
			if err != nil {
				return err
			}
			return printJSON(tk)
		},
	}
	return cmd
}

func newTaskStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "stop <id>",
		Short:       "Stop a task",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := task.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			if err := svc.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"stopped": args[0]})
		},
	}
}

func newTaskListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			tasks, err := store.ListTasks(cmd.Context())
			if err != nil {
				return err
			}
			return printJSON(tasks)
		},
	}
}

func newTaskTellCmd() *cobra.Command {
	var taskFlag, toMemberID string

	cmd := &cobra.Command{
		Use:         "tell <content>",
		Short:       "Tell a teammate",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, fromMemberID, err := resolveTaskContext(taskFlag)
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := task.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			if err := svc.Send(cmd.Context(), taskID, fromMemberID, toMemberID, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "delivered",
				"from":   fromMemberID,
				"to":     toMemberID,
			})
		},
	}
	cmd.Flags().StringVar(&taskFlag, "task", "", "Task ID (defaults to CLIER_TASK_ID)")
	cmd.Flags().StringVar(&toMemberID, "to", "", "Recipient member ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newTaskNoteCmd() *cobra.Command {
	var taskFlag string

	cmd := &cobra.Command{
		Use:         "note <content>",
		Short:       "Post a progress note",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, memberID, err := resolveTaskContext(taskFlag)
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := task.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			if err := svc.Note(cmd.Context(), taskID, memberID, args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "posted",
				"member": memberID,
				"task":   taskID,
			})
		},
	}
	cmd.Flags().StringVar(&taskFlag, "task", "", "Task ID (defaults to CLIER_TASK_ID)")
	return cmd
}

func newTaskNotesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "notes <task-id>",
		Short: "List task notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			notes, err := store.ListNotesByTaskID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printJSON(notes)
		},
	}
}

func newTaskAttachCmd() *cobra.Command {
	var memberFlag string

	cmd := &cobra.Command{
		Use:   "attach <task-id>",
		Short: "Attach to a running task's terminal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			term := terminal.NewTmuxTerminal(store)

			var memberID *string
			if memberFlag != "" {
				memberID = &memberFlag
			}
			return term.Attach(args[0], memberID)
		},
	}
	cmd.Flags().StringVar(&memberFlag, "member", "", "Attach to a specific member's window")
	return cmd
}

// resolveTaskContext resolves task ID and member ID from env vars set by clier.
// CLIER_TASK_ID identifies the task, CLIER_MEMBER_ID identifies the sender.
func resolveTaskContext(taskFlag string) (taskID, memberID string, err error) {
	taskID = taskFlag
	if taskID == "" {
		taskID = os.Getenv("CLIER_TASK_ID")
	}
	if taskID == "" {
		return "", "", errors.New("--task flag or CLIER_TASK_ID must be set")
	}
	memberID = os.Getenv("CLIER_MEMBER_ID")
	return taskID, memberID, nil
}
