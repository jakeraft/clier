package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
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

	tmpDir, err := os.MkdirTemp("", "clier-dashboard-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	if err := copyEmbedDir(distFS, distRoot, tmpDir); err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}

	indexPath := filepath.Join(tmpDir, "index.html")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read index.html: %w", err)
	}

	original := string(indexBytes)
	injected := strings.Replace(original, jsonPlaceholder, string(jsonBytes), 1)
	if injected == original {
		return fmt.Errorf("placeholder %q not found in index.html", jsonPlaceholder)
	}

	if err := os.WriteFile(indexPath, []byte(injected), 0644); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}

	fmt.Printf("Dashboard: %s\n", indexPath)
	return exec.Command("open", indexPath).Run()
}

func copyEmbedDir(fsys embed.FS, root, dest string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		content, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, content, 0644)
	})
}
