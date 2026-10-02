package main

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
)

// writeFakePrettier writes a shell script at path that records its argv to
// "<path>.args", prints output, and exits with exitCode — the tool mock every
// run test points PATH (or node_modules/.bin) at.
func writeFakePrettier(t *testing.T, path, output string, exitCode int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"$0.args\"\n" +
		"cat <<'FITNESS_EOF'\n" + output + "\nFITNESS_EOF\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// recordedArgv reads the argv the fake at path captured.
func recordedArgv(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path + ".args")
	if err != nil {
		t.Fatalf("fake prettier was not invoked: %v", err)
	}
	var argv []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line != "" {
			argv = append(argv, line)
		}
	}
	return argv
}

// writeFiles creates each rel:content pair under root.
func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const sharedCfgToken = "%CFG%"

// embeddedCfgToken stands in for the materialized embedded config path in
// expected argv (the cache path is only known at runtime).
const embeddedCfgToken = "%EMBEDDED%"

// materializedConfig returns the embedded shared config's materialized path
// for filename — what the check must fall back to with no local config and no
// node_modules install.
func materializedConfig(t *testing.T, filename string) string {
	t.Helper()
	dir, err := sharedconf.Materialize()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, filename)
}

const cleanOutput = "Checking formatting...\nAll matched files use Prettier code style!"

// flagsToken stands in for the flags that make Prettier ignore nothing it
// can parse.
const flagsToken = "%FLAGS%"

// installedCfgRel is the npm-era shared config location.
var installedCfgRel = filepath.Join("node_modules", "@mayjournal", "fitness-shared", "config", prettierConfigCjs)

// runCase is one TestRun scenario.
type runCase struct {
	name       string
	files      map[string]string
	staged     string
	args       []string
	output     string
	exit       int
	sharedCfg  bool
	wantOk     bool
	wantErrors []string
	wantFiles  int
	wantArgv   []string // nil: Prettier must not run
}

// setupRun builds the case's repo and a fake Prettier on PATH, returning
// the repo root and the fake's path.
func setupRun(t *testing.T, tc runCase) (root, fake string) {
	t.Helper()
	root = t.TempDir()
	// A package.json satisfies the no-JS-project guard; a case that
	// supplies its own (e.g. a prettier field) overwrites this one.
	writeFiles(t, root, map[string]string{"package.json": "{}"})
	writeFiles(t, root, tc.files)
	if tc.sharedCfg {
		writeFiles(t, root, map[string]string{installedCfgRel: "module.exports = {};"})
	}
	binDir := t.TempDir()
	fake = filepath.Join(binDir, "prettier")
	writeFakePrettier(t, fake, tc.output, tc.exit)
	t.Setenv("PATH", binDir+":/usr/bin:/bin")
	t.Setenv("FITNESS_CHANGED_FILES", tc.staged)
	return root, fake
}

// expandArgv replaces the tokens in want with their runtime values.
func expandArgv(t *testing.T, root string, want []string) []string {
	t.Helper()
	tokens := map[string][]string{
		sharedCfgToken:   {filepath.Join(root, installedCfgRel)},
		embeddedCfgToken: {materializedConfig(t, prettierConfigCjs)},
		flagsToken:       {"--ignore-unknown", "--ignore-path", ignoreNothing, "--with-node-modules"},
	}
	var out []string
	for _, a := range want {
		if values, ok := tokens[a]; ok {
			out = append(out, values...)
			continue
		}
		out = append(out, a)
	}
	return out
}

// assertArgv checks what the fake recorded, or that it never ran.
func assertArgv(t *testing.T, root, fake string, want []string) {
	t.Helper()
	if want == nil {
		if _, err := os.Stat(fake + ".args"); err == nil {
			t.Fatal("fake prettier was invoked, want it skipped")
		}
		return
	}
	if got, exp := recordedArgv(t, fake), expandArgv(t, root, want); !reflect.DeepEqual(got, exp) {
		t.Fatalf("argv = %v, want %v", got, exp)
	}
}

// assertVerdict compares a run's result with the case's expectations.
func assertVerdict(t *testing.T, res checkkit.Result, err error, tc runCase) {
	t.Helper()
	if err != nil || res.Ok != tc.wantOk || res.FilesChecked != tc.wantFiles {
		t.Fatalf("ok = %v, files = %d, want %v, %d (err %v, errors %v)", res.Ok, res.FilesChecked, tc.wantOk, tc.wantFiles, err, res.Errors)
	}
	if !slices.Equal(res.Errors, tc.wantErrors) {
		t.Fatalf("errors = %v, want %v", res.Errors, tc.wantErrors)
	}
}

func TestRun(t *testing.T) {
	cases := []runCase{
		{
			name:      "unscoped run checks every file when the repo is clean",
			files:     map[string]string{".prettierrc.json": "{}"},
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 2,
			wantArgv:  []string{flagsToken, "--check", ".prettierrc.json", "package.json"},
		},
		{
			name:      "installed shared config wins when consumer has none",
			sharedCfg: true,
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 2,
			wantArgv:  []string{"--config", sharedCfgToken, flagsToken, "--check", filepath.ToSlash(installedCfgRel), "package.json"},
		},
		{
			name:      "embedded config materializes when nothing is installed",
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 1,
			wantArgv:  []string{"--config", embeddedCfgToken, flagsToken, "--check", "package.json"},
		},
		{
			name:      "consumer package.json prettier field suppresses shared config",
			files:     map[string]string{"package.json": `{"prettier":"./x.cjs"}`},
			sharedCfg: true,
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 2,
			wantArgv:  []string{flagsToken, "--check", filepath.ToSlash(installedCfgRel), "package.json"},
		},
		{
			name:  "warn lines become per-file errors",
			files: map[string]string{".prettierrc.json": "{}"},
			output: "Checking formatting...\n[warn] src/foo.ts\n" +
				"[warn] Code style issues found in 1 file above. Run Prettier with --write to fix.",
			exit:       1,
			wantErrors: []string{"src/foo.ts"},
			wantFiles:  1,
			wantArgv:   []string{flagsToken, "--check", ".prettierrc.json", "package.json"},
		},
		{
			name:       "fallback message when non-zero exit parses nothing",
			files:      map[string]string{".prettierrc.json": "{}"},
			output:     "Prettier failed",
			exit:       2,
			wantErrors: []string{prettierFallbackMessage},
			wantFiles:  2,
			wantArgv:   []string{flagsToken, "--check", ".prettierrc.json", "package.json"},
		},
		{
			name: "every changed file still on disk goes to Prettier",
			files: map[string]string{
				".prettierrc.json":    "{}",
				"bar.ts":              "x",
				"go/go.mod":           "module x",
				"notes.mdc":           "x",
				".gitignore":          "x",
				"githooks/commit-msg": "x",
				"packages/a.ts":       "x",
			},
			staged:    "bar.ts\ngo/go.mod\nnotes.mdc\n.gitignore\ngithooks/commit-msg\nmissing.ts\npackages/a.ts",
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 6,
			wantArgv: []string{flagsToken, "--check", "bar.ts", "go/go.mod", "notes.mdc", ".gitignore",
				"githooks/commit-msg", "packages/a.ts"},
		},
		{
			name:   "changed list of deleted files skips prettier",
			files:  map[string]string{".prettierrc.json": "{}"},
			staged: "missing.ts",
			output: cleanOutput,
			wantOk: true,
		},
		{
			name:      "passthrough args replace check mode",
			files:     map[string]string{".prettierrc.json": "{}"},
			args:      []string{"--write", "."},
			wantOk:    true,
			wantFiles: 2,
			wantArgv:  []string{flagsToken, "--write", "."},
		},
		{
			name:      "passthrough keeps the shared config",
			sharedCfg: true,
			args:      []string{"--write", "."},
			wantOk:    true,
			wantFiles: 2,
			wantArgv:  []string{"--config", sharedCfgToken, flagsToken, "--write", "."},
		},
		{
			name:       "a .prettierignore fails before Prettier runs",
			files:      map[string]string{".prettierrc.json": "{}", ".prettierignore": "dist\n"},
			wantErrors: []string{".prettierignore exists; Prettier checks every tracked file, so delete it and fix the findings instead"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, fake := setupRun(t, tc)
			res, err := run(root, tc.args)
			assertVerdict(t, res, err, tc)
			assertArgv(t, root, fake, tc.wantArgv)
		})
	}
}

func TestRunMissingPrettier(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"package.json": "{}"})
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FITNESS_CHANGED_FILES", "")
	res, err := run(root, nil)
	assertVerdict(t, res, err, runCase{wantErrors: []string{notInstalled}})
}

// TestPrettierIgnoreFailsWithoutPackageJSON pins that a .prettierignore
// fails even in a repo Prettier would otherwise skip.
func TestPrettierIgnoreFailsWithoutPackageJSON(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{".prettierignore": "x\n"})
	t.Setenv("FITNESS_CHANGED_FILES", "")
	res, err := run(root, nil)
	assertVerdict(t, res, err, runCase{wantErrors: []string{
		".prettierignore exists; Prettier checks every tracked file, so delete it and fix the findings instead"}})
}

// TestRunSkipsWhenNoPackageJSON pins the self-gating guard: a repo with no
// package.json passes without running (or requiring) Prettier.
func TestRunSkipsWhenNoPackageJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FITNESS_CHANGED_FILES", "")
	res, err := run(root, nil)
	assertVerdict(t, res, err, runCase{wantOk: true})
}

func TestNodeModulesBinPreferredOverPath(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{".prettierrc.json": "{}", "package.json": "{}"})
	local := filepath.Join(root, "node_modules", ".bin", "prettier")
	writeFakePrettier(t, local, cleanOutput, 0)
	binDir := t.TempDir()
	writeFakePrettier(t, filepath.Join(binDir, "prettier"), "[warn] wrong-binary.ts", 1)
	t.Setenv("PATH", binDir+":/usr/bin:/bin")
	t.Setenv("FITNESS_CHANGED_FILES", "")
	res, err := run(root, nil)
	// The walk outside git sees the fake and its argv record too.
	assertVerdict(t, res, err, runCase{wantOk: true, wantFiles: 3})
	assertArgv(t, root, local, []string{flagsToken, "--check", ".prettierrc.json", "node_modules/.bin/prettier", "package.json"})
}

func TestParsePrettierOutput(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "extracts file paths from warn lines",
			output: "[warn] a.ts\n[warn] b.ts\n[warn] Code style issues found in 2 files above.",
			want:   []string{"a.ts", "b.ts"},
		},
		{
			name:   "trims whitespace around warn lines and paths",
			output: "  [warn] c.ts  \n",
			want:   []string{"c.ts"},
		},
		{
			name:   "ignores non-warn output",
			output: "Checking formatting...\nAll matched files use Prettier code style!",
			want:   nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parsePrettierOutput(tc.output); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parsePrettierOutput = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHasPrettierConfig(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  bool
	}{
		{"prettierrc json file", map[string]string{".prettierrc.json": "{}"}, true},
		{"prettier config ts file", map[string]string{"prettier.config.ts": "export default {};"}, true},
		{"package.json prettier string", map[string]string{"package.json": `{"prettier":"./x.cjs"}`}, true},
		{"package.json prettier object", map[string]string{"package.json": `{"prettier":{"semi":true}}`}, true},
		{"package.json prettier null", map[string]string{"package.json": `{"prettier":null}`}, false},
		{"package.json invalid json", map[string]string{"package.json": "not valid json"}, false},
		{"empty dir", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, tc.files)
			if got := hasPrettierConfig(root); got != tc.want {
				t.Fatalf("hasPrettierConfig = %v, want %v", got, tc.want)
			}
		})
	}
}

// writePluginProbePrettier writes a fake Prettier that, like the real one with
// the shared config, starts only when NODE_PATH reaches the packagejson plugin.
func writePluginProbePrettier(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"for d in $(echo \"$NODE_PATH\" | tr ':' ' '); do\n" +
		"  if [ -d \"$d/prettier-plugin-packagejson\" ]; then echo 'All matched files use Prettier code style!'; exit 0; fi\n" +
		"done\n" +
		"echo \"[error] Cannot find module 'prettier-plugin-packagejson'\"\n" +
		"exit 2\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestSharedConfigFindsRepoPlugins covers a repo on the shared config: with
// the plugins installed it passes, and without them Prettier's own error is
// reported beside the fallback message.
func TestSharedConfigFindsRepoPlugins(t *testing.T) {
	t.Setenv("FITNESS_CHANGED_FILES", "")
	t.Setenv("NODE_PATH", "")
	t.Setenv("PATH", "/usr/bin:/bin")
	cases := []struct {
		name  string
		files map[string]string
		want  runCase
	}{
		{"plugins installed", map[string]string{"node_modules/prettier-plugin-packagejson/package.json": "{}"},
			runCase{wantOk: true, wantFiles: 3}},
		{"plugins missing", nil, runCase{wantFiles: 2, wantErrors: []string{
			prettierFallbackMessage, "[error] Cannot find module 'prettier-plugin-packagejson'"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{"package.json": "{}"})
			writeFiles(t, root, tc.files)
			writePluginProbePrettier(t, filepath.Join(root, "node_modules", ".bin", "prettier"))
			res, err := run(root, nil)
			assertVerdict(t, res, err, tc.want)
		})
	}
}

// nodePathEntries splits the last env entry, which must be NODE_PATH.
func nodePathEntries(t *testing.T, env []string) []string {
	t.Helper()
	value, ok := strings.CutPrefix(env[len(env)-1], "NODE_PATH=")
	if !ok {
		t.Fatalf("last env entry = %q, want NODE_PATH", env[len(env)-1])
	}
	return strings.Split(value, string(os.PathListSeparator))
}

func TestWithNodePath(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"node_modules/x/package.json": "{}"})
	got := withNodePath([]string{"A=1", "NODE_PATH=/elsewhere"}, root)
	entries := nodePathEntries(t, got)
	if !strings.HasSuffix(entries[0], "node_modules") || entries[len(entries)-1] != "/elsewhere" {
		t.Fatalf("NODE_PATH = %v, want the repo's node_modules first and /elsewhere kept", entries)
	}
	if !slices.Equal(got[:len(got)-1], []string{"A=1"}) {
		t.Fatalf("env = %v, want A=1 kept and one NODE_PATH", got)
	}
	if empty := withNodePath([]string{"A=1"}, t.TempDir()); len(empty) != 1 {
		t.Fatalf("no node_modules: env = %v, want unchanged", empty)
	}
}
