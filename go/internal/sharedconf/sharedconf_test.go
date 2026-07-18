package sharedconf

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// embeddedNames are the six frozen config files this package ships.
var embeddedNames = []string{
	"constants.cjs",
	"cspell.json",
	"eslint.base.mjs",
	"eslint.config.mjs",
	"prettier.config.cjs",
	"vitest.config.mjs",
}

// redirectCache points materialization at a scratch cache for one test (the
// tests never touch the real user cache) and returns the scratch base.
func redirectCache(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	userCacheDir = func() (string, error) { return base, nil }
	t.Cleanup(func() { userCacheDir = os.UserCacheDir })
	return base
}

// writeTree creates each rel:content pair under base.
func writeTree(t *testing.T, base string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFileNamesMatchTheFrozenSet(t *testing.T) {
	if got := fileNames(); !reflect.DeepEqual(got, embeddedNames) {
		t.Fatalf("fileNames() = %v, want %v", got, embeddedNames)
	}
}

func TestMaterializeWritesTheWholeDirectory(t *testing.T) {
	base := redirectCache(t)
	dir, err := Materialize()
	if err != nil {
		t.Fatal(err)
	}
	if parent := filepath.Dir(dir); parent != filepath.Join(base, "fitness") {
		t.Fatalf("cache parent = %q, want %q", parent, filepath.Join(base, "fitness"))
	}
	name := filepath.Base(dir)
	if !strings.HasPrefix(name, "sharedconf-") || len(name) != len("sharedconf-")+12 {
		t.Fatalf("cache dir name = %q, want sharedconf-<12 hex chars>", name)
	}
	for _, fname := range embeddedNames {
		p := filepath.Join(dir, fname)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("%s not materialized: %v", fname, err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("%s mode = %v, want 0644", fname, info.Mode().Perm())
		}
		want, err := fs.ReadFile(configFS, path.Join(configDir, fname))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s content differs from the embedded copy", fname)
		}
	}
}

func TestMaterializeIdempotentAndRestoresMissingFiles(t *testing.T) {
	redirectCache(t)
	dir, err := Materialize()
	if err != nil {
		t.Fatal(err)
	}
	// An existing file is trusted (the directory name is content-keyed) and
	// never rewritten; a missing file is restored.
	kept := filepath.Join(dir, "cspell.json")
	if err := os.WriteFile(kept, []byte("locally kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	restored := filepath.Join(dir, "prettier.config.cjs")
	if err := os.Remove(restored); err != nil {
		t.Fatal(err)
	}
	again, err := Materialize()
	if err != nil {
		t.Fatal(err)
	}
	if again != dir {
		t.Fatalf("second Materialize = %q, want the same dir %q", again, dir)
	}
	if got, err := os.ReadFile(kept); err != nil || string(got) != "locally kept" {
		t.Fatalf("existing file rewritten: %q, %v", got, err)
	}
	want, _ := fs.ReadFile(configFS, path.Join(configDir, "prettier.config.cjs"))
	if got, err := os.ReadFile(restored); err != nil || string(got) != string(want) {
		t.Fatalf("missing file not restored: %v", err)
	}
}

func TestMaterializeConcurrent(t *testing.T) {
	redirectCache(t)
	const writers = 8
	dirs := make([]string, writers)
	errs := make([]error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dirs[i], errs[i] = Materialize()
		}()
	}
	wg.Wait()
	for i := range writers {
		if errs[i] != nil {
			t.Fatalf("writer %d: %v", i, errs[i])
		}
		if dirs[i] != dirs[0] {
			t.Fatalf("writer %d dir = %q, want %q", i, dirs[i], dirs[0])
		}
	}
	for _, fname := range embeddedNames {
		want, _ := fs.ReadFile(configFS, path.Join(configDir, fname))
		got, err := os.ReadFile(filepath.Join(dirs[0], fname))
		if err != nil || string(got) != string(want) {
			t.Fatalf("%s corrupted by concurrent materialization: %v", fname, err)
		}
	}
}

func TestMaterializeFallsBackToTempDir(t *testing.T) {
	userCacheDir = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { userCacheDir = os.UserCacheDir })
	scratch := t.TempDir()
	t.Setenv("TMPDIR", scratch)
	dir, err := Materialize()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(os.TempDir(), "fitness"); filepath.Dir(dir) != want {
		t.Fatalf("fallback cache parent = %q, want %q", filepath.Dir(dir), want)
	}
	if _, err := os.Stat(filepath.Join(dir, "cspell.json")); err != nil {
		t.Fatalf("fallback materialization incomplete: %v", err)
	}
}

func TestResolve(t *testing.T) {
	t.Run("repo-local file wins", func(t *testing.T) {
		redirectCache(t)
		root := t.TempDir()
		writeTree(t, root, map[string]string{
			"cspell.json": "{}",
			filepath.ToSlash(filepath.Join(installedRel, "cspell.json")): "{}",
		})
		if got, want := Resolve(root, "cspell.json"), filepath.Join(root, "cspell.json"); got != want {
			t.Fatalf("Resolve = %q, want local %q", got, want)
		}
	})
	t.Run("installed package wins over embedded, walking up", func(t *testing.T) {
		redirectCache(t)
		base := t.TempDir()
		installed := filepath.Join(base, installedRel, "prettier.config.cjs")
		writeTree(t, base, map[string]string{
			filepath.ToSlash(filepath.Join(installedRel, "prettier.config.cjs")): "module.exports = {};",
		})
		root := filepath.Join(base, "nested", "app")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if got := Resolve(root, "prettier.config.cjs"); got != installed {
			t.Fatalf("Resolve = %q, want installed %q", got, installed)
		}
	})
	t.Run("install missing the file falls through to embedded", func(t *testing.T) {
		redirectCache(t)
		root := t.TempDir()
		writeTree(t, root, map[string]string{
			filepath.ToSlash(filepath.Join(installedRel, "cspell.json")): "{}",
		})
		got := Resolve(root, "prettier.config.cjs")
		dir, err := Materialize()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(dir, "prettier.config.cjs"); got != want {
			t.Fatalf("Resolve = %q, want materialized %q", got, want)
		}
	})
	t.Run("no config anywhere materializes the embedded copy", func(t *testing.T) {
		redirectCache(t)
		got := Resolve(t.TempDir(), "eslint.config.mjs")
		if base := filepath.Base(got); base != "eslint.config.mjs" {
			t.Fatalf("Resolve = %q, want an eslint.config.mjs path", got)
		}
		if !strings.Contains(got, "sharedconf-") {
			t.Fatalf("Resolve = %q, want a materialized cache path", got)
		}
		if _, err := os.Stat(got); err != nil {
			t.Fatalf("resolved path not on disk: %v", err)
		}
	})
}

func TestResolveDir(t *testing.T) {
	t.Run("installed config directory wins, walking up", func(t *testing.T) {
		redirectCache(t)
		base := t.TempDir()
		writeTree(t, base, map[string]string{
			filepath.ToSlash(filepath.Join(installedRel, "vitest.config.mjs")): "export default {};",
		})
		root := filepath.Join(base, "nested", "app")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if got, want := ResolveDir(root), filepath.Join(base, installedRel); got != want {
			t.Fatalf("ResolveDir = %q, want installed %q", got, want)
		}
	})
	t.Run("no install materializes the embedded copy", func(t *testing.T) {
		redirectCache(t)
		got := ResolveDir(t.TempDir())
		dir, err := Materialize()
		if err != nil {
			t.Fatal(err)
		}
		if got != dir {
			t.Fatalf("ResolveDir = %q, want materialized %q", got, dir)
		}
		if _, err := os.Stat(filepath.Join(got, "vitest.config.mjs")); err != nil {
			t.Fatalf("materialized dir incomplete: %v", err)
		}
	})
}
