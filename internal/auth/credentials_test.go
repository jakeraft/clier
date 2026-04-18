package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	creds := &Credentials{Token: "test-token", Login: "jakeraft"}
	if err := Save(path, creds); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Token != "test-token" || loaded.Login != "jakeraft" {
		t.Fatalf("got %+v", loaded)
	}

	// Check file permission
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("got perm %o, want 0600", info.Mode().Perm())
	}
}

func TestLoad_NotExists(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error")
	}
	var fault *domain.Fault
	if !errors.As(err, &fault) || fault.Kind != domain.KindAuthRequired {
		t.Fatalf("got %v, want auth_required fault", err)
	}
}

func TestLoad_ReadFailureIsNotDowngradedToAuthRequired(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	var fault *domain.Fault
	if errors.As(err, &fault) && fault.Kind == domain.KindAuthRequired {
		t.Fatalf("read failure should not be downgraded to auth_required: %v", err)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	_ = Save(path, &Credentials{Token: "t", Login: "l"})
	if err := Delete(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("file should be deleted")
	}
}

func TestDelete_NotExists(t *testing.T) {
	if err := Delete("/nonexistent/path"); err != nil {
		t.Fatal(err)
	}
}
