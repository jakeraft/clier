package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

// readSource reads bytes from a local file or an HTTP(S) URL.
func readSource(src string) ([]byte, error) {
	if isURL(src) {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(src)
		if err != nil {
			return nil, fmt.Errorf("fetch URL: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch URL: %s", resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	}
	return os.ReadFile(src)
}

// isURL returns true if the source looks like an HTTP(S) URL.
func isURL(src string) bool {
	return strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")
}

// isLocalDir returns true if the path exists and is a directory.
func isLocalDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// basePath returns the directory portion of a source path/URL.
func basePath(src string) string {
	if isURL(src) {
		idx := strings.LastIndex(src, "/")
		if idx >= 0 {
			return src[:idx]
		}
		return src
	}
	return filepath.Dir(src)
}

// joinPath joins a base path/URL with a relative file name.
func joinPath(base, name string) string {
	if isURL(base) {
		return base + "/" + name
	}
	return filepath.Join(base, name)
}

// indexFile represents an index.json manifest that lists envelope files.
type indexFile struct {
	Files []string `json:"files"`
}

func init() {
	rootCmd.AddCommand(newImportCmd())
}

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file-or-url-or-dir>",
		Short: "Import entities from a JSON envelope, index.json manifest, or directory",
		Long: `Import entities from a JSON envelope file, an index.json manifest, or a directory.

  - Envelope file (has "type" field): imports a single entity
  - index.json (has "files" field): imports all listed files in order
  - Directory or URL without index.json: auto-discovers index.json inside`,
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]

			// If it's a local directory, append /index.json.
			if !isURL(src) && isLocalDir(src) {
				src = joinPath(src, "index.json")
			}

			data, err := readSource(src)
			if err != nil {
				// For URLs that don't end with index.json, try appending it.
				if isURL(src) && !strings.HasSuffix(src, "/index.json") {
					src = strings.TrimRight(src, "/") + "/index.json"
					data, err = readSource(src)
				}
				if err != nil {
					return err
				}
			}

			// Try to detect the JSON shape.
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				return fmt.Errorf("parse JSON: %w", err)
			}

			client := newAPIClient()
			owner := resolveOwner()

			// If the JSON has a "files" array, treat it as an index manifest.
			if _, ok := raw["files"]; ok {
				var idx indexFile
				if err := json.Unmarshal(data, &idx); err != nil {
					return fmt.Errorf("parse index.json: %w", err)
				}
				if len(idx.Files) == 0 {
					return errors.New("index.json has no files listed")
				}
				base := basePath(src)
				for _, f := range idx.Files {
					fileSrc := joinPath(base, f)
					fileData, err := readSource(fileSrc)
					if err != nil {
						return fmt.Errorf("read %s: %w", fileSrc, err)
					}
					if err := importEnvelope(client, owner, fileData); err != nil {
						return fmt.Errorf("import %s: %w", f, err)
					}
				}
				return nil
			}

			// Otherwise treat it as a single envelope.
			return importEnvelope(client, owner, data)
		},
	}
}

// importEnvelope imports a single JSON envelope via the API.
func importEnvelope(client *api.Client, owner string, data []byte) error {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("parse envelope: %w", err)
	}

	switch envelope.Type {
	case "claude_md":
		resp, err := client.CreateClaudeMd(owner, json.RawMessage(envelope.Data))
		if err != nil {
			return err
		}
		return printJSON(resp)

	case "skill":
		resp, err := client.CreateSkill(owner, json.RawMessage(envelope.Data))
		if err != nil {
			return err
		}
		return printJSON(resp)

	case "claude_settings":
		resp, err := client.CreateClaudeSettings(owner, json.RawMessage(envelope.Data))
		if err != nil {
			return err
		}
		return printJSON(resp)

	case "member":
		resp, err := client.CreateMember(owner, json.RawMessage(envelope.Data))
		if err != nil {
			return err
		}
		return printJSON(resp)

	case "team":
		resp, err := client.ImportTeam(owner, json.RawMessage(envelope.Data))
		if err != nil {
			return err
		}
		return printJSON(resp)

	default:
		return fmt.Errorf("unknown envelope type: %q", envelope.Type)
	}
}
