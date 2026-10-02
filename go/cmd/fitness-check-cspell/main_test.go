package main

// cspell:ignore borwn nteh quik xqzzt vbnmm skipdir zzzqqqv mispeled wrod

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
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
	t.Setenv("FITNESS_CHANGED_FILES", strings.Join(files, "\n"))
}

func TestDescribe(t *testing.T) {
	if check.Describe.Name != "cspell" {
		t.Fatalf("name = %q, want cspell", check.Describe.Name)
	}
	if check.Describe.TimeoutMs != 0 {
		t.Fatalf("timeout = %d, want 0 (runner default)", check.Describe.TimeoutMs)
	}
}

// result is the verdict a case expects: ok, files checked, and the leading
// error lines.
type result struct {
	ok         bool
	checked    int
	wantErrors []string
}

// assertResult compares a run's verdict with want.
func assertResult(t *testing.T, res checkkit.Result, err error, want result) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok != want.ok || res.FilesChecked != want.checked {
		t.Fatalf("ok=%v files=%d, want ok=%v files=%d (errors: %v)",
			res.Ok, res.FilesChecked, want.ok, want.checked, res.Errors)
	}
	assertLeadingErrors(t, res.Errors, want.wantErrors)
}

// assertLeadingErrors checks that got starts with want, in order.
func assertLeadingErrors(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) < len(want) || !slices.Equal(got[:len(want)], want) {
		t.Fatalf("errors = %v, want them to start with %v", got, want)
	}
}

// runCase writes files into a fresh root, sets the changed-file scope, and
// runs the check there.
func runCase(t *testing.T, files map[string]string, staged []string) (checkkit.Result, error) {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		write(t, root, rel, content)
	}
	setStaged(t, staged...)
	return run(root, nil)
}

func TestStagedMode(t *testing.T) {
	cases := []struct {
		name   string
		files  map[string]string
		staged []string
		want   result
	}{
		{
			name:   "missing files drop to a clean empty run",
			staged: []string{"missing.md"},
			want:   result{ok: true},
		},
		{
			name:   "clean staged file passes",
			files:  map[string]string{"docs.md": "clean readable words\n"},
			staged: []string{"docs.md"},
			want:   result{ok: true, checked: 1},
		},
		{
			name:   "misspelling fails with cspell's error format",
			files:  map[string]string{"sub/s.md": "good words\nteh quik borwn fox\n"},
			staged: []string{"sub/s.md"},
			want: result{checked: 1, wantErrors: []string{
				"sub/s.md:2:5 - Unknown word (quik)",
				"sub/s.md:2:10 - Unknown word (borwn)",
			}},
		},
		{
			name:   "project words win over the base dictionaries",
			files:  map[string]string{"cspell.json": `{"words":["borwn"]}`, "s.md": "borwn fox\n"},
			staged: []string{"s.md"},
			want:   result{ok: true, checked: 1},
		},
		{
			name:   "binary staged file is counted but never spelled",
			files:  map[string]string{"blob.bin": "xqzzt\x00vbnmm"},
			staged: []string{"blob.bin"},
			want:   result{ok: true, checked: 1},
		},
		{
			name: "lock, ignore, and dictionary files are spelled like any other",
			files: map[string]string{
				".gitignore":                      "borwn\n",
				"package-lock.json":               `{"name":"zzzqqqv"}`,
				"go/internal/spell/dict/node.txt": "skipdir\n",
			},
			staged: []string{".gitignore", "go/internal/spell/dict/node.txt", "package-lock.json"},
			want: result{checked: 3, wantErrors: []string{
				".gitignore:1:1 - Unknown word (borwn)",
				"go/internal/spell/dict/node.txt:1:1 - Unknown word (skipdir)",
				"package-lock.json:1:10 - Unknown word (zzzqqqv)",
			}},
		},
		{
			name: "mixed source shapes stay clean",
			files: map[string]string{
				"main.go":      "package main\n\nfunc main() {\n\tprintln(\"hello world\\nagain\")\n}\n",
				"index.ts":     "export const camelCaseName = { enabled: true };\n",
				"package.json": `{"name":"fixture","scripts":{"test":"vitest run"}}`,
			},
			staged: []string{"main.go", "index.ts", "package.json"},
			want:   result{ok: true, checked: 3},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := runCase(t, tc.files, tc.staged)
			assertResult(t, res, err, tc.want)
		})
	}
}

func writeBody(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestBodyMode covers cspell's body mode: an explicit --body-file document is
// spell-checked with config resolved from root, so a project word declared in
// the root's cspell.json is accepted while genuine misspellings fail. Using a
// temp-dir cspell.json keeps the test independent of any checkout path.
// "borwn" is unknown to the base dictionaries; accepting it proves the
// project cspell.json resolved from root.
func TestBodyMode(t *testing.T) {
	root := t.TempDir()
	write(t, root, "cspell.json", `{"words":["borwn"]}`)
	cases := []struct {
		name, body string
		want       result
	}{
		{"misspellings flagged", "teh mispeled wrod in the description\n",
			result{checked: 1, wantErrors: []string{"(description):1:5 - Unknown word (mispeled)"}}},
		{"project word accepted via root config", "The borwn value is intentional here.\n", result{ok: true, checked: 1}},
		{"clean pass", "This describes a clean readable change to the project.\n", result{ok: true, checked: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := run(root, []string{"--body-file", writeBody(t, tc.body)})
			assertResult(t, res, err, tc.want)
		})
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestUnscopedModeChecksEveryTrackedFile proves the unscoped run spells
// every tracked file of any type, gitignored or under node_modules, and
// leaves out only untracked files and those marked linguist-generated.
func TestUnscopedModeChecksEveryTrackedFile(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q")
	files := map[string]string{
		".gitignore":                   "generated/\n",
		".gitattributes":               "*.lock linguist-generated\n",
		"deps/pkg.lock":                "zzzqqqv\n",
		"README.md":                    "clean words\n",
		"b-docs/inner.md":              "teh quik fox\n",
		"generated/out.md":             "borwn\n",
		"node_modules/pkg/vendored.md": "skipdir\n",
		"notes.txt":                    "vbnmm\n",
		"untracked.md":                 "xqzzt\n",
	}
	for rel, content := range files {
		write(t, root, rel, content)
	}
	git(t, root, "add", "-f", ".gitignore", ".gitattributes", "deps", "README.md", "b-docs", "generated", "node_modules", "notes.txt")
	setStaged(t)
	res, err := run(root, nil)
	assertResult(t, res, err, result{checked: 7, wantErrors: []string{
		"b-docs/inner.md:1:5 - Unknown word (quik)",
		"generated/out.md:1:1 - Unknown word (borwn)",
		"node_modules/pkg/vendored.md:1:1 - Unknown word (skipdir)",
		"notes.txt:1:1 - Unknown word (vbnmm)",
	}})
}

func TestIgnorePathsFails(t *testing.T) {
	for _, config := range []string{`{"ignorePaths":["skipdir"]}`, `{"ignorePaths":[]}`} {
		res, err := runCase(t, map[string]string{"cspell.json": config, "a.md": "clean words\n"}, nil)
		assertResult(t, res, err, result{wantErrors: []string{
			"cspell.json sets ignorePaths; cspell checks every tracked file, so remove ignorePaths and fix the findings instead",
		}})
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
		setStaged(t, "doc.md")
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
		setStaged(t, "doc.md")
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
		setStaged(t, "doc.md")
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
		setStaged(t, "doc.md")
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Ok || len(res.Errors) == 0 {
			t.Fatalf("expected misspellings flagged under the embedded config: %+v", res)
		}
	})
}
