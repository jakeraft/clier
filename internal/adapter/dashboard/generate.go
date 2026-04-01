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

// Generate writes dashboard.html to dataDir. Does not open the browser.
func Generate(ctx context.Context, store *db.Store, dataDir string, distFS embed.FS, distRoot string) error {
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

	outPath := filepath.Join(dataDir, "dashboard.html")
	if err := os.WriteFile(outPath, []byte(injected), 0644); err != nil {
		return fmt.Errorf("write dashboard.html: %w", err)
	}

	return nil
}

// Open generates dashboard.html and opens it in the browser.
func Open(ctx context.Context, store *db.Store, dataDir string, distFS embed.FS, distRoot string) error {
	if err := Generate(ctx, store, dataDir, distFS, distRoot); err != nil {
		return err
	}

	outPath := filepath.Join(dataDir, "dashboard.html")
	fmt.Printf("Dashboard: %s\n", outPath)
	return exec.Command("open", outPath).Run()
}
