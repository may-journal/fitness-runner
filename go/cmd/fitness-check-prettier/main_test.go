package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
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

const cleanOutput = "Checking formatting...\nAll matched files use Prettier code style!"

func TestRun(t *testing.T) {
	cases := []struct {
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
		wantArgv   []string
	}{
		{
			name:      "glob mode passes when repo is clean",
			files:     map[string]string{".prettierrc.json": "{}"},
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 0,
			wantArgv:  []string{"--check", "."},
		},
		{
			name:      "shared config used when consumer has none",
			sharedCfg: true,
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 0,
			wantArgv:  []string{"--config", sharedCfgToken, "--check", "."},
		},
		{
			name:      "consumer package.json prettier field suppresses shared config",
			files:     map[string]string{"package.json": `{"prettier":"./x.cjs"}`},
			sharedCfg: true,
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 0,
			wantArgv:  []string{"--check", "."},
		},
		{
			name:  "warn lines become per-file errors",
			files: map[string]string{".prettierrc.json": "{}"},
			output: "Checking formatting...\n[warn] src/foo.ts\n" +
				"[warn] Code style issues found in 1 file above. Run Prettier with --write to fix.",
			exit:       1,
			wantOk:     false,
			wantErrors: []string{"src/foo.ts"},
			wantFiles:  1,
			wantArgv:   []string{"--check", "."},
		},
		{
			name:       "fallback message when non-zero exit parses nothing",
			files:      map[string]string{".prettierrc.json": "{}"},
			output:     "Prettier failed",
			exit:       2,
			wantOk:     false,
			wantErrors: []string{prettierFallbackMessage},
			wantFiles:  0,
			wantArgv:   []string{"--check", "."},
		},
		{
			name: "staged paths pass through the skip filters",
			files: map[string]string{
				".prettierrc.json":    "{}",
				"bar.ts":              "x",
				"go/go.mod":           "module x",
				"notes.mdc":           "x",
				".gitignore":          "x",
				"githooks/commit-msg": "x",
				"scripts/x.sh":        "x",
				"packages/a.ts":       "x",
			},
			staged: "bar.ts\ngo/go.mod\nnotes.mdc\n.gitignore\ngithooks/commit-msg\n" +
				"scripts/x.sh\nmissing.ts\npackages/a.ts",
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 2,
			wantArgv:  []string{"--check", "bar.ts", "packages/a.ts"},
		},
		{
			name:      "staged list filtered to nothing falls back to glob",
			files:     map[string]string{".prettierrc.json": "{}", "go/go.mod": "module x"},
			staged:    "go/go.mod",
			output:    cleanOutput,
			wantOk:    true,
			wantFiles: 0,
			wantArgv:  []string{"--check", "."},
		},
		{
			name:      "passthrough args replace check mode",
			files:     map[string]string{".prettierrc.json": "{}"},
			args:      []string{"--write", "."},
			output:    "",
			wantOk:    true,
			wantFiles: 0,
			wantArgv:  []string{"--write", "."},
		},
		{
			name:      "passthrough keeps the shared config",
			sharedCfg: true,
			args:      []string{"--write", "."},
			output:    "",
			wantOk:    true,
			wantFiles: 0,
			wantArgv:  []string{"--config", sharedCfgToken, "--write", "."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, tc.files)
			cfgPath := filepath.Join(root, "node_modules", "@mayjournal", "fitness-shared", "config", prettierConfigCjs)
			if tc.sharedCfg {
				writeFiles(t, root, map[string]string{
					filepath.Join("node_modules", "@mayjournal", "fitness-shared", "config", prettierConfigCjs): "module.exports = {};",
				})
			}
			binDir := t.TempDir()
			fake := filepath.Join(binDir, "prettier")
			writeFakePrettier(t, fake, tc.output, tc.exit)
			t.Setenv("PATH", binDir+":/usr/bin:/bin")
			t.Setenv("FITNESS_STAGED_FILES", tc.staged)

			res, err := run(root, tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.wantOk {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.wantOk, res.Errors)
			}
			if len(tc.wantErrors) > 0 && !reflect.DeepEqual(res.Errors, tc.wantErrors) {
				t.Fatalf("errors = %v, want %v", res.Errors, tc.wantErrors)
			}
			if tc.wantOk && len(res.Errors) != 0 {
				t.Fatalf("errors = %v, want none", res.Errors)
			}
			if res.FilesChecked != tc.wantFiles {
				t.Fatalf("filesChecked = %d, want %d", res.FilesChecked, tc.wantFiles)
			}
			want := make([]string, len(tc.wantArgv))
			for i, a := range tc.wantArgv {
				if a == sharedCfgToken {
					a = cfgPath
				}
				want[i] = a
			}
			if got := recordedArgv(t, fake); !reflect.DeepEqual(got, want) {
				t.Fatalf("argv = %v, want %v", got, want)
			}
		})
	}
}

func TestRunMissingPrettier(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FITNESS_STAGED_FILES", "")
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 0 || len(res.Errors) != 1 || res.Errors[0] != notInstalled {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestNodeModulesBinPreferredOverPath(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{".prettierrc.json": "{}"})
	local := filepath.Join(root, "node_modules", ".bin", "prettier")
	writeFakePrettier(t, local, cleanOutput, 0)
	binDir := t.TempDir()
	writeFakePrettier(t, filepath.Join(binDir, "prettier"), "[warn] wrong-binary.ts", 1)
	t.Setenv("PATH", binDir+":/usr/bin:/bin")
	t.Setenv("FITNESS_STAGED_FILES", "")
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected node_modules/.bin fake to run and pass, got %+v", res)
	}
	if got := recordedArgv(t, local); !reflect.DeepEqual(got, []string{"--check", "."}) {
		t.Fatalf("argv = %v, want [--check .]", got)
	}
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
