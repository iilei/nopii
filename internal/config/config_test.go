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

func TestLoadConfigClassifiers(t *testing.T) {
	path := filepath.Join(t.TempDir(), configpkg.FileName)
	content := []byte(
		"version = 1\n" +
			"[classifiers.username]\n" +
			"label = \"USER\"\n" +
			"pattern = '''(?m)(?:^|[[:space:]])@([A-Za-z0-9_-]+)'''\n",
	)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := configpkg.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := cfg.Classifiers["username"]; !ok || got.Label != "USER" || got.Pattern == "" {
		t.Fatalf("expected classifier username mapping, got %#v", cfg.Classifiers)
	}
}
