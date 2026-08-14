package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverNearestConfig(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(home, "nopii-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, FileName)
	if err := os.WriteFile(want, []byte("version=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(child)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
