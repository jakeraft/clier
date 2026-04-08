package cmd

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Envelope is the generic export/import wrapper for any entity.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func init() {
	rootCmd.AddCommand(newExportCmd())
}

func newExportCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "export <id>",
		Short: "Export an entity by UUID to JSON envelope",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			ctx := cmd.Context()

			// Try each entity type until found.
			type probe struct {
				typeName string
				fetch    func() (any, error)
			}
			probes := []probe{
				{"team", func() (any, error) { t, e := store.GetTeam(ctx, id); return t, e }},
				{"member", func() (any, error) { m, e := store.GetMember(ctx, id); return m, e }},
				{"claude_md", func() (any, error) { cm, e := store.GetClaudeMd(ctx, id); return cm, e }},
				{"skill", func() (any, error) { sk, e := store.GetSkill(ctx, id); return sk, e }},
				{"claude_settings", func() (any, error) { st, e := store.GetClaudeSettings(ctx, id); return st, e }},
				{"claude_json", func() (any, error) { cj, e := store.GetClaudeJson(ctx, id); return cj, e }},
			}

			for _, p := range probes {
				entity, err := p.fetch()
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						continue
					}
					return fmt.Errorf("fetch %s: %w", p.typeName, err)
				}

				dataBytes, err := json.Marshal(entity)
				if err != nil {
					return fmt.Errorf("marshal %s: %w", p.typeName, err)
				}

				envelope := Envelope{
					Type: p.typeName,
					Data: dataBytes,
				}

				out, err := json.MarshalIndent(envelope, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal envelope: %w", err)
				}

				if output != "" {
					if err := os.WriteFile(output, append(out, '\n'), 0644); err != nil {
						return fmt.Errorf("write file: %w", err)
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Exported %s to %s\n", p.typeName, output)
					return nil
				}

				out = append(out, '\n')
				_, err = cmd.OutOrStdout().Write(out)
				return err
			}

			return fmt.Errorf("entity not found: %s", id)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: stdout)")
	return cmd
}
