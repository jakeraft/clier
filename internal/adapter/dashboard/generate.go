package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/db"
)

const jsonPlaceholder = "/* JSON_DATA */"

// Generate writes dashboard.html to outPath and returns the file path.
func Generate(ctx context.Context, store *db.Store, outPath string, distFS embed.FS, distRoot string) (string, error) {
	data, err := Collect(ctx, store)
	if err != nil {
		return "", fmt.Errorf("collect data: %w", err)
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal data: %w", err)
	}

	indexBytes, err := distFS.ReadFile(filepath.Join(distRoot, "index.html"))
	if err != nil {
		return "", fmt.Errorf("read embedded index.html: %w", err)
	}

	original := string(indexBytes)
	injected := strings.Replace(original, jsonPlaceholder, string(jsonBytes), 1)
	if injected == original {
		return "", fmt.Errorf("placeholder %q not found in index.html", jsonPlaceholder)
	}

	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	if err := os.WriteFile(outPath, []byte(injected), 0644); err != nil {
		return "", fmt.Errorf("write dashboard.html: %w", err)
	}

	return outPath, nil
}
