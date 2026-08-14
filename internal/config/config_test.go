package config_test

import (
	"os"
	"path/filepath"
	"testing"

	configpkg "github.com/iilei/nopii/internal/config"
)

func TestDiscoverNearestConfig(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Fatalf("remove %s: %v", root, err)
		}
	})
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0o750); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, configpkg.FileName)
	if err := os.WriteFile(want, []byte("version=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := configpkg.Discover(child)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
