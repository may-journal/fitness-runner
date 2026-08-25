package main

// cspell:ignore borwn nteh quik xqzzt vbnmm skipdir zzzqqqv

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/spell"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setStaged(t *testing.T, files ...string) {
	t.Helper()
	t.Setenv("FITNESS_STAGED_FILES", strings.Join(files, "\n"))
}

func TestDescribe(t *testing.T) {
	if check.Describe.Name != "cspell" {
		t.Fatalf("name = %q, want cspell", check.Describe.Name)
	}
	if check.Describe.TimeoutMs != 0 {
		t.Fatalf("timeout = %d, want 0 (runner default)", check.Describe.TimeoutMs)
	}
}

func TestStagedMode(t *testing.T) {
	cases := []struct {
		name       string
		files      map[string]string
		staged     []string
		ok         bool
		checked    int
		wantErrors []string
	}{
		{
			name:    "always-ignored basenames and missing files drop to a clean empty run",
			files:   map[string]string{".gitignore": "borwn\n", "tsconfig.json": "{", "package-lock.json": "{"},
			staged:  []string{".gitignore", "tsconfig.json", "package-lock.json", "missing.md"},
			ok:      true,
			checked: 0,
		},
		{
			name:    "clean staged file passes",
			files:   map[string]string{"docs.md": "clean readable words\n"},
			staged:  []string{"docs.md"},
			ok:      true,
			checked: 1,
		},
		{
			name:    "misspelling fails with cspell's error format",
			files:   map[string]string{"sub/s.md": "good words\nteh quik borwn fox\n"},
			staged:  []string{"sub/s.md"},
			ok:      false,
			checked: 1,
			wantErrors: []string{
				"sub/s.md:2:5 - Unknown word (quik)",
				"sub/s.md:2:10 - Unknown word (borwn)",
			},
		},
		{
			name:    "project words win over the base dictionaries",
			files:   map[string]string{"cspell.json": `{"words":["borwn"]}`, "s.md": "borwn fox\n"},
			staged:  []string{"s.md"},
			ok:      true,
			checked: 1,
		},
		{
			name:    "binary staged file is counted but never spelled",
			files:   map[string]string{"blob.bin": "xqzzt\x00vbnmm"},
			staged:  []string{"blob.bin"},
			ok:      true,
			checked: 1,
		},
		{
			// The embedded shared cspell.json ignores go/internal/spell/dict;
			// staged mode must honor ignorePaths so a staged dictionary source
			// file is never spell-checked. The fixture content would fail if
			// scanned (zzzqqqv), proving the skip rather than a clean pass.
			name:    "staged file under an ignorePaths directory is skipped",
			files:   map[string]string{"go/internal/spell/dict/node.txt": "zzzqqqv\n"},
			staged:  []string{"go/internal/spell/dict/node.txt"},
			ok:      true,
			checked: 0,
		},
		{
			name: "mixed source shapes stay clean",
			files: map[string]string{
				"main.go":      "package main\n\nfunc main() {\n\tprintln(\"hello world\\nagain\")\n}\n",
				"index.ts":     "export const camelCaseName = { enabled: true };\n",
				"package.json": `{"name":"fixture","scripts":{"test":"vitest run"}}`,
			},
			staged:  []string{"main.go", "index.ts", "package.json"},
			ok:      true,
			checked: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tc.files {
				write(t, root, rel, content)
			}
			setStaged(t, tc.staged...)
			res, err := run(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok || res.FilesChecked != tc.checked {
				t.Fatalf("ok=%v files=%d, want ok=%v files=%d (errors: %v)",
					res.Ok, res.FilesChecked, tc.ok, tc.checked, res.Errors)
			}
			for i, want := range tc.wantErrors {
				if i >= len(res.Errors) || res.Errors[i] != want {
					t.Fatalf("errors = %v, want %v", res.Errors, tc.wantErrors)
				}
			}
		})
	}
}

func TestGlobModeWalksIgnorePathsAndSorts(t *testing.T) {
	root := t.TempDir()
	write(t, root, "cspell.json", `{"words":["borwn"],"ignorePaths":["skipdir","**/*.test.md"]}`)
	write(t, root, "README.md", "borwn is a project word\n")
	write(t, root, "b-docs/inner.md", "teh quik fox\n")
	write(t, root, "a.test.md", "zzzqqqv misspelled but ignored\n")
	write(t, root, "skipdir/skipped.md", "zzzqqqv misspelled but ignored\n")
	write(t, root, "node_modules/pkg/vendored.md", "zzzqqqv always ignored\n")
	write(t, root, "notes.txt", "not markdown zzzqqqv\n")
	setStaged(t)
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 2 {
		t.Fatalf("ok=%v files=%d, want fail with 2 files (errors: %v)", res.Ok, res.FilesChecked, res.Errors)
	}
	want := []string{"b-docs/inner.md:1:5 - Unknown word (quik)"}
	if len(res.Errors) != 1 || res.Errors[0] != want[0] {
		t.Fatalf("errors = %v, want %v", res.Errors, want)
	}
}

func TestGlobModeUsesGitignore(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v (%s)", err, out)
		}
	}
	write(t, root, "cspell.json", `{"useGitignore":true,"ignorePaths":[]}`)
	write(t, root, ".gitignore", "generated/\n")
	write(t, root, "kept.md", "teh quik fox\n")
	write(t, root, "generated/out.md", "zzzqqqv would fail if scanned\n")
	setStaged(t)
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesChecked != 1 {
		t.Fatalf("filesChecked = %d, want 1 (gitignored dir excluded); errors: %v", res.FilesChecked, res.Errors)
	}
	if res.Ok || !strings.Contains(res.Errors[0], "Unknown word (quik)") {
		t.Fatalf("expected quik flagged in kept.md, got %v", res.Errors)
	}
}

// installedCspellJSON is the compat location an npm-era install resolves
// through — it must keep winning over the embedded fallback.
const installedCspellJSON = "node_modules/@mayjournal/fitness-shared/config/cspell.json"

func TestConfigResolution(t *testing.T) {
	shared := `{"words":["zzzqqqv"]}`
	t.Run("installed shared package wins over embedded", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, installedCspellJSON, shared)
		write(t, root, "doc.md", "zzzqqqv allowed by shared config\n")
		setStaged(t)
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Ok || res.FilesChecked != 1 {
			t.Fatalf("installed shared config not honored: %+v", res)
		}
	})
	t.Run("root cspell.json wins over installed", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "cspell.json", `{"words":[]}`)
		write(t, root, installedCspellJSON, shared)
		write(t, root, "doc.md", "zzzqqqv no longer allowed\n")
		setStaged(t)
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Ok {
			t.Fatalf("expected shared words to be ignored when root config exists: %+v", res)
		}
	})
	t.Run("no config anywhere materializes the embedded shared config", func(t *testing.T) {
		// "mayjournal" is in the embedded shared cspell.json words list but
		// not in the base dictionaries — assert the discrimination first, so
		// a pass below can only come from the materialized config.
		if base := spell.NewEmbeddedChecker(); len(base.CheckText("mayjournal")) != 1 {
			t.Fatal("mayjournal must be unknown to the base dictionaries for this test to prove anything")
		}
		root := t.TempDir()
		write(t, root, "doc.md", "mayjournal is allowed by the embedded shared config\n")
		setStaged(t)
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Ok || res.FilesChecked != 1 {
			t.Fatalf("embedded shared config not honored: %+v", res)
		}
	})
	t.Run("embedded fallback still flags real misspellings", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "doc.md", "teh quik borwn fox\n")
		setStaged(t)
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Ok || len(res.Errors) == 0 {
			t.Fatalf("expected misspellings flagged under the embedded config: %+v", res)
		}
	})
}
