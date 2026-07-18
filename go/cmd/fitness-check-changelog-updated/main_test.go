package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// inject pins the staged diff and the clock for one test.
func inject(t *testing.T, diff string, at time.Time) {
	t.Helper()
	prevDiff, prevNow := stagedDiff, now
	stagedDiff = func(string) string { return diff }
	now = func() time.Time { return at }
	t.Cleanup(func() { stagedDiff, now = prevDiff, prevNow })
}

func TestRun(t *testing.T) {
	fixed := time.Date(2026, 2, 16, 14, 30, 0, 0, time.Local)
	cases := []struct {
		name         string
		staged       string // FITNESS_STAGED_FILES value
		diff         string
		changelog    bool   // create CHANGELOG.md at root
		pkgVersion   string // create package.json with this version
		at           time.Time
		ok           bool
		filesChecked int
		wantErrors   []string
	}{
		{
			name:         "empty staged env passes with zero files",
			staged:       "",
			changelog:    true,
			at:           fixed,
			ok:           true,
			filesChecked: 0,
		},
		{
			name:         "only changelog staged passes with nothing to compare",
			staged:       "CHANGELOG.md",
			diff:         "+++ b/CHANGELOG.md\n+ - item\n",
			changelog:    true,
			at:           fixed,
			ok:           true,
			filesChecked: 1,
		},
		{
			name:   "three shared words pass",
			staged: "src/foo.ts\nCHANGELOG.md",
			diff: "+++ b/src/foo.ts\n+ Added new runner feature for validation.\n" +
				"+ Export runner from index.\n+++ b/CHANGELOG.md\n" +
				"+ - Added new runner feature; validation export.\n",
			changelog:    true,
			at:           fixed,
			ok:           true,
			filesChecked: 1,
		},
		{
			name:         "missing changelog fails",
			staged:       "src/bar.ts",
			diff:         "+ feature code\n",
			at:           fixed,
			ok:           false,
			filesChecked: 1,
			wantErrors:   []string{msgChangelogMissing},
		},
		{
			name:   "fewer than three shared words fails with suggestion",
			staged: "src/baz.ts\nCHANGELOG.md",
			diff: "+++ b/src/baz.ts\n+ New feature runner validation export helper.\n" +
				"+++ b/CHANGELOG.md\n+ - Minor fix.\n",
			changelog:    true,
			at:           fixed,
			ok:           false,
			filesChecked: 1,
			wantErrors: []string{
				"CHANGELOG.md additions should mention at least 3 words from your staged changes (found 0: )",
				"e.g. use words like: new, feature, runner, validation, export, helper",
			},
		},
		{
			name:         "changelog not staged fails",
			staged:       "src/bar.ts",
			diff:         "+++ b/src/bar.ts\n+ New feature code here.\n",
			changelog:    true,
			at:           fixed,
			ok:           false,
			filesChecked: 1,
			wantErrors:   []string{msgStageChangelog},
		},
		{
			name:         "nested changelog path counts as rest not changelog",
			staged:       "packages/x/CHANGELOG.md",
			diff:         "+++ b/packages/x/CHANGELOG.md\n+ New words here friend.\n",
			changelog:    true,
			at:           fixed,
			ok:           false,
			filesChecked: 1,
			wantErrors:   []string{msgStageChangelog},
		},
		{
			name:   "heading matching current time passes",
			staged: "src/foo.ts\nCHANGELOG.md",
			diff: "+++ b/src/foo.ts\n+ New runner feature.\n+++ b/CHANGELOG.md\n" +
				"+ ### 2026.02.16.1430\n+ - New runner feature.\n",
			changelog:    true,
			at:           fixed,
			ok:           true,
			filesChecked: 1,
		},
		{
			name:   "heading matching package version suffix passes",
			staged: "src/foo.ts\nCHANGELOG.md",
			diff: "+++ b/src/foo.ts\n+ New runner feature.\n+++ b/CHANGELOG.md\n" +
				"+ ### 2026.02.16.1430\n+ - New runner feature.\n",
			changelog:    true,
			pkgVersion:   "0.1.0-2026.02.16.1430",
			at:           time.Date(2026, 2, 16, 19, 0, 0, 0, time.Local),
			ok:           true,
			filesChecked: 1,
		},
		{
			name:   "heading not matching current time fails",
			staged: "src/foo.ts\nCHANGELOG.md",
			diff: "+++ b/src/foo.ts\n+ New runner feature.\n+++ b/CHANGELOG.md\n" +
				"+ ### 2026.02.16.1900\n+ - New runner feature.\n",
			changelog:    true,
			at:           fixed,
			ok:           false,
			filesChecked: 1,
			wantErrors:   []string{msgChangelogTime + " (expected ### 2026.02.16.1430)"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.changelog {
				writeFile(t, dir, "CHANGELOG.md", "# Changelog\n\n### 2026-02-15\n\n- item\n")
			}
			if tc.pkgVersion != "" {
				writeFile(t, dir, "package.json", `{"version":"`+tc.pkgVersion+`"}`)
			}
			t.Setenv("FITNESS_STAGED_FILES", tc.staged)
			inject(t, tc.diff, tc.at)
			res, err := run(dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok || res.FilesChecked != tc.filesChecked {
				t.Fatalf("ok=%v filesChecked=%d, want ok=%v filesChecked=%d (errors: %v)",
					res.Ok, res.FilesChecked, tc.ok, tc.filesChecked, res.Errors)
			}
			if tc.wantErrors != nil && !reflect.DeepEqual(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %q, want %q", res.Errors, tc.wantErrors)
			}
		})
	}
}

func TestParseStagedDiff(t *testing.T) {
	diff := "diff --git a/x b/x\n--- a/src/a.ts\n+++ b/src/a.ts\n" +
		"+first line\n++ not an addition\n+ second\n" +
		"+++ /dev/null\n+dropped\n" +
		"+++ b/src/a.ts\n+third\n"
	got := parseStagedDiff(diff)
	want := []diffFile{
		{path: "src/a.ts", added: "first line  second third"},
		{path: "/dev/null", added: "dropped"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseStagedDiff = %+v, want %+v", got, want)
	}
}

func TestParseStagedDiffIgnoresLinesBeforeFirstHeader(t *testing.T) {
	if got := parseStagedDiff("+early\n+lines\n"); len(got) != 0 {
		t.Fatalf("parseStagedDiff = %+v, want empty", got)
	}
}

func TestExtractWords(t *testing.T) {
	got := extractWords("Add-runner, add RUNNER! v2 2026.02.16 ab abc")
	want := []string{"add", "runner", "2026", "abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractWords = %q, want %q", got, want)
	}
}

// TestStandaloneAgainstRealGitRepo covers the no-env path: staged files and
// the diff both come from git itself.
func TestStandaloneAgainstRealGitRepo(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)
	writeFile(t, dir, "CHANGELOG.md", "# Changelog\n")
	writeFile(t, dir, "src.txt", "placeholder\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "init")
	unsetStagedEnv(t)

	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 0 {
		t.Fatalf("nothing staged: got %+v, want pass with 0 files", res)
	}

	writeFile(t, dir, "CHANGELOG.md", "# Changelog\n\n- added runner feature validation\n")
	writeFile(t, dir, "src.txt", "added runner feature validation code\n")
	git(t, dir, "add", "-A")

	res, err = run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("staged overlap: got %+v, want pass with 1 file", res)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
}

// unsetStagedEnv removes FITNESS_STAGED_FILES entirely (t.Setenv first so the
// original value is restored after the test).
func unsetStagedEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FITNESS_STAGED_FILES", "")
	if err := os.Unsetenv("FITNESS_STAGED_FILES"); err != nil {
		t.Fatal(err)
	}
}
