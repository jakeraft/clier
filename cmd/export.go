package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
			client := newAPIClient()
			owner := resolveOwner()

			// Try each entity type until found.
			type probe struct {
				typeName string
				fetch    func() (any, error)
			}
			probes := []probe{
				{"team", func() (any, error) { return client.GetTeam(owner, id) }},
				{"member", func() (any, error) { return client.GetMember(owner, id) }},
				{"claude_md", func() (any, error) { return client.GetClaudeMd(owner, id) }},
				{"skill", func() (any, error) { return client.GetSkill(owner, id) }},
				{"claude_settings", func() (any, error) { return client.GetClaudeSettings(owner, id) }},
			}

			for _, p := range probes {
				entity, err := p.fetch()
				if err != nil {
					// API returns 404 for not found; try next type.
					if strings.Contains(err.Error(), "api error 404") {
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
