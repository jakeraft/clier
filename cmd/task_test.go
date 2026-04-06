package cmd

import (
	"os"
	"testing"
)

func TestReadContent_FromArgs(t *testing.T) {
	got, err := readContent([]string{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}

func TestReadContent_FromStdin(t *testing.T) {
	r, w, _ := os.Pipe()
	w.WriteString("from stdin\n")
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	got, err := readContent(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "from stdin" {
		t.Fatalf("got %q, want %q", got, "from stdin")
	}
}

func TestReadContent_DashMeansStdin(t *testing.T) {
	r, w, _ := os.Pipe()
	w.WriteString("via dash\n")
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	got, err := readContent([]string{"-"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "via dash" {
		t.Fatalf("got %q, want %q", got, "via dash")
	}
}

func TestReadContent_EmptyStdinError(t *testing.T) {
	r, w, _ := os.Pipe()
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	_, err := readContent(nil)
	if err == nil {
		t.Fatal("expected error for empty stdin")
	}
}
