package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	db "github.com/jakeraft/clier/internal/adapter/db"
)

const jsonPlaceholder = "/* JSON_DATA */"

func Open(ctx context.Context, store *db.Store, distFS embed.FS, distRoot string) error {
	data, err := Collect(ctx, store)
	if err != nil {
		return fmt.Errorf("collect data: %w", err)
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	indexBytes, err := distFS.ReadFile(filepath.Join(distRoot, "index.html"))
	if err != nil {
		return fmt.Errorf("read embedded index.html: %w", err)
	}

	original := string(indexBytes)
	injected := strings.Replace(original, jsonPlaceholder, string(jsonBytes), 1)
	if injected == original {
		return fmt.Errorf("placeholder %q not found in index.html", jsonPlaceholder)
	}

	tmpFile, err := os.CreateTemp("", "clier-dashboard-*.html")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tmpFile.WriteString(injected); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	fmt.Printf("Dashboard: %s\n", tmpFile.Name())
	return exec.Command("open", tmpFile.Name()).Run()
}
