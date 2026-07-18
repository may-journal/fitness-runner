package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
)

// sortKeysJSON is a realistic JSON-formatter payload with one sort-keys
// finding (note the single quotes the shell fake must survive).
const sortKeysJSON = `[{"errorCount":1,"filePath":"/r/bad.js","messages":[{"column":19,"line":1,"message":"Expected object keys to be in natural ascending case-sensitive order. 'a' should be before 'z'.","ruleId":"sort-keys","severity":2}],"warningCount":0}]`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeScript installs an executable /bin/sh script at path.
func writeScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// fakeEslint installs a fake eslint in root/node_modules/.bin that emits the
// given JSON via heredoc (quote-safe) and exits with code.
func fakeEslint(t *testing.T, root, stdout string, exitCode int) {
	t.Helper()
	// /bin/cat by absolute path: the fake runs under a scrubbed PATH.
	body := "/bin/cat <<'FIXTURE'\n" + stdout + "\nFIXTURE\nexit " + strconv.Itoa(exitCode)
	writeScript(t, filepath.Join(root, "node_modules", ".bin", "eslint"), body)
}

// isolate scrubs PATH and the staged-file env so no real tool or ambient
// runner context leaks into a test.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FITNESS_STAGED_FILES", "")
}

func TestRunJudgment(t *testing.T) {
	cases := []struct {
		name     string
		stdout   string
		exitCode int
		ok       bool
		files    int
		errors   []string
	}{
		{"clean run passes", "[]", 0, true, 0, nil},
		{"finding fails with formatted line", sortKeysJSON, 1, false, 1, []string{
			"/r/bad.js:1:19 - Expected object keys to be in natural ascending case-sensitive order. 'a' should be before 'z'. (sort-keys)",
		}},
		{"exit from errorCount not process exit", sortKeysJSON, 0, false, 1, []string{
			"/r/bad.js:1:19 - Expected object keys to be in natural ascending case-sensitive order. 'a' should be before 'z'. (sort-keys)",
		}},
		{"ignore notice filtered but file counted",
			`[{"errorCount":0,"filePath":"/r/skip.js","messages":[{"message":"File ignored because of a matching ignore pattern. Use \"--no-ignore\" to disable file ignore settings or use \"--no-warn-ignored\" to suppress this warning.","ruleId":null,"severity":1}],"warningCount":1}]`,
			0, true, 1, nil},
		{"warning without errorCount still fails",
			`[{"errorCount":0,"filePath":"/r/warn.js","messages":[{"column":2,"line":3,"message":"Watch out.","ruleId":"sort-keys","severity":1}],"warningCount":1}]`,
			0, false, 1, []string{"/r/warn.js:3:2 - Watch out. (sort-keys)"}},
		{"null ruleId omits suffix",
			`[{"errorCount":1,"filePath":"/r/bad.ts","messages":[{"message":"Parsing error: boom.","ruleId":null}],"warningCount":0}]`,
			1, false, 1, []string{"/r/bad.ts:0:0 - Parsing error: boom."}},
		{"garbage stdout falls back", "not json", 1, false, 0, []string{
			eslintFallbackMessage, "Output: not json",
		}},
		{"object stdout falls back", "{}", 0, false, 0, []string{
			eslintFallbackMessage, "Output: {}",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolate(t)
			root := t.TempDir()
			fakeEslint(t, root, tc.stdout, tc.exitCode)
			res, err := run(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok || res.FilesChecked != tc.files {
				t.Fatalf("ok=%v files=%d, want ok=%v files=%d (errors: %v)",
					res.Ok, res.FilesChecked, tc.ok, tc.files, res.Errors)
			}
			want := tc.errors
			if want == nil {
				want = []string{}
			}
			if !reflect.DeepEqual(res.Errors, want) {
				t.Fatalf("errors = %#v, want %#v", res.Errors, want)
			}
		})
	}
}

func TestRunCrashFallbackQuotesStderr(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	writeScript(t, filepath.Join(root, "node_modules", ".bin", "eslint"),
		"echo 'Error: Could not find config file.' >&2\nexit 2")
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{eslintFallbackMessage, "Output: Error: Could not find config file."}
	if res.Ok || res.FilesChecked != 0 || !reflect.DeepEqual(res.Errors, want) {
		t.Fatalf("result = %+v, want errors %#v", res, want)
	}
}

func TestRunMissingBinary(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 0 || len(res.Errors) != 1 || res.Errors[0] != missingEslintMessage {
		t.Fatalf("unexpected result: %+v", res)
	}
}

// installedConfigMjs is the compat path an npm-era @mayjournal/fitness-shared
// install serves the shared flat config from.
var installedConfigMjs = filepath.Join("node_modules", "@mayjournal", "fitness-shared", "config", eslintConfigMjs)

// invokeAndRecord runs the check against root with a fake eslint that records
// its working directory and argv, returning the recorded lines.
func invokeAndRecord(t *testing.T, root string) []string {
	t.Helper()
	argsFile := filepath.Join(t.TempDir(), "args")
	writeScript(t, filepath.Join(root, "node_modules", ".bin", "eslint"),
		"pwd -P > "+argsFile+"\nprintf '%s\\n' \"$@\" >> "+argsFile+"\necho '[]'")
	if _, err := run(root, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

// wantInvocation is the full expected recording: eslint's cwd, then the
// argv with the given --config path.
func wantInvocation(t *testing.T, root, config string) []string {
	t.Helper()
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return []string{realRoot, "--config", config, "--no-error-on-unmatched-pattern", "--format", "json", "."}
}

func TestRunInvocation(t *testing.T) {
	t.Run("installed shared config wins over embedded", func(t *testing.T) {
		isolate(t)
		root := t.TempDir()
		writeFile(t, filepath.Join(root, installedConfigMjs), "export default [];")
		got := invokeAndRecord(t, root)
		want := wantInvocation(t, root, filepath.Join(root, installedConfigMjs))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("invocation = %#v, want %#v", got, want)
		}
	})
	t.Run("repo-local config wins over installed", func(t *testing.T) {
		isolate(t)
		root := t.TempDir()
		writeFile(t, filepath.Join(root, eslintConfigMjs), "export default [];")
		writeFile(t, filepath.Join(root, installedConfigMjs), "export default [];")
		got := invokeAndRecord(t, root)
		want := wantInvocation(t, root, filepath.Join(root, eslintConfigMjs))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("invocation = %#v, want %#v", got, want)
		}
	})
	t.Run("embedded config materializes when nothing else resolves", func(t *testing.T) {
		isolate(t)
		root := t.TempDir()
		got := invokeAndRecord(t, root)
		dir, err := sharedconf.Materialize()
		if err != nil {
			t.Fatal(err)
		}
		want := wantInvocation(t, root, filepath.Join(dir, eslintConfigMjs))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("invocation = %#v, want %#v", got, want)
		}
		if _, err := os.Stat(filepath.Join(dir, eslintConfigMjs)); err != nil {
			t.Fatalf("materialized config missing: %v", err)
		}
	})
}

func TestRunStagedPaths(t *testing.T) {
	isolate(t)
	root := t.TempDir()
	quoted := `src/bar "quoted".ts`
	for _, p := range []string{"a.js", quoted} {
		writeFile(t, filepath.Join(root, p), "x")
	}
	t.Setenv("FITNESS_STAGED_FILES", "a.js\n"+quoted+"\nREADME.md")
	argsFile := filepath.Join(t.TempDir(), "args")
	writeScript(t, filepath.Join(root, "node_modules", ".bin", "eslint"),
		"printf '%s\\n' \"$@\" > "+argsFile+"\necho '[]'")
	if _, err := run(root, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	patterns := got[len(got)-2:]
	if !reflect.DeepEqual(patterns, []string{"a.js", quoted}) {
		t.Fatalf("patterns = %#v, want the two staged lintable files", patterns)
	}
}

func TestPathsToLint(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"a.js", "b.cjs", "c.mjs", "d.ts", "e.tsx", "f.d.ts", "g.test.ts", "h.spec.js"} {
		writeFile(t, filepath.Join(root, p), "x")
	}
	cases := []struct {
		name   string
		staged []string
		want   []string
	}{
		{"no staged lints dot", nil, []string{"."}},
		{"lintable staged kept", []string{"a.js", "b.cjs", "c.mjs", "d.ts", "e.tsx"},
			[]string{"a.js", "b.cjs", "c.mjs", "d.ts", "e.tsx"}},
		{"declaration and test files dropped", []string{"a.js", "f.d.ts", "g.test.ts", "h.spec.js"},
			[]string{"a.js"}},
		{"non-lintable extensions dropped", []string{"a.js", "README.md", "x.go"}, []string{"a.js"}},
		{"missing file dropped", []string{"a.js", "gone.ts"}, []string{"a.js"}},
		{"all filtered falls back to dot", []string{"f.d.ts", "README.md", "gone.ts"}, []string{"."}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathsToLint(root, tc.staged); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("pathsToLint = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestFormatMessage(t *testing.T) {
	cases := []struct {
		name string
		msg  eslintMessage
		want string
	}{
		{"with rule", eslintMessage{Column: 2, Line: 1, Message: "x", RuleID: "no-var"}, "/f.ts:1:2 - x (no-var)"},
		{"missing fields zeroed, no rule suffix", eslintMessage{Message: "y"}, "/f.ts:0:0 - y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatMessage("/f.ts", tc.msg); got != tc.want {
				t.Fatalf("formatMessage = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseResults(t *testing.T) {
	cases := []struct {
		name  string
		input string
		ok    bool
		count int
	}{
		{"object is not an array", "{}", false, 0},
		{"invalid json", "not json", false, 0},
		{"null is not an array", "null", false, 0},
		{"empty array", "[]", true, 0},
		{"one result", `[{"errorCount":0,"filePath":"/a.ts","messages":[],"warningCount":0}]`, true, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, ok := parseResults([]byte(tc.input))
			if ok != tc.ok || len(results) != tc.count {
				t.Fatalf("parseResults = (%d results, %v), want (%d, %v)", len(results), ok, tc.count, tc.ok)
			}
		})
	}
}

// TestRunInvocationWalksUpForInstalledConfig pins the compat walk: a nested
// package still finds an ancestor's npm-era install.
func TestRunInvocationWalksUpForInstalledConfig(t *testing.T) {
	isolate(t)
	base := t.TempDir()
	writeFile(t, filepath.Join(base, installedConfigMjs), "export default [];")
	root := filepath.Join(base, "nested", "pkg")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	got := invokeAndRecord(t, root)
	want := wantInvocation(t, root, filepath.Join(base, installedConfigMjs))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invocation = %#v, want %#v", got, want)
	}
}

func TestFindEslint(t *testing.T) {
	t.Run("node_modules bin in ancestor wins", func(t *testing.T) {
		isolate(t)
		base := t.TempDir()
		bin := filepath.Join(base, "node_modules", ".bin", "eslint")
		writeScript(t, bin, "echo '[]'")
		root := filepath.Join(base, "nested", "pkg")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		got, found := findEslint(root)
		if !found || got != bin {
			t.Fatalf("findEslint = (%q, %v), want %q", got, found, bin)
		}
	})
	t.Run("falls back to PATH", func(t *testing.T) {
		isolate(t)
		pathDir := t.TempDir()
		bin := filepath.Join(pathDir, "eslint")
		writeScript(t, bin, "echo '[]'")
		t.Setenv("PATH", pathDir)
		got, found := findEslint(t.TempDir())
		if !found || got != bin {
			t.Fatalf("findEslint = (%q, %v), want %q", got, found, bin)
		}
	})
	t.Run("missing everywhere", func(t *testing.T) {
		isolate(t)
		if got, found := findEslint(t.TempDir()); found {
			t.Fatalf("findEslint = %q, want not found", got)
		}
	})
}
