package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fullThresholdsJS is a raw config the threshold fast path accepts.
const fullThresholdsJS = "module.exports = { test: { coverage: { thresholds: " +
	"{ branches: 100, functions: 100, lines: 100, statements: 100 } } } };"

// partialThresholdsJS fails the gate: branches at 90.
const partialThresholdsJS = "module.exports = { test: { coverage: { thresholds: " +
	"{ branches: 90, functions: 100, lines: 100, statements: 100 } } } };"

// fullThresholdsPkgJSON passes the gate without any vitest.config.* file.
const fullThresholdsPkgJSON = `{"vitest":{"coverage":{"thresholds":` +
	`{"branches":100,"functions":100,"lines":100,"statements":100}}}}`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// installFakeVitest writes an executable vitest script into dir/vitest that
// prints out/err, records its argv to argsFile when non-empty, and exits
// with code.
func installFakeVitest(t *testing.T, dir, out, errOut string, code int, argsFile string) {
	t.Helper()
	script := "#!/bin/sh\n"
	if argsFile != "" {
		script += "printf '%s\\n' \"$@\" > \"" + argsFile + "\"\n"
	}
	if out != "" {
		script += "printf '%s' '" + out + "'\n"
	}
	if errOut != "" {
		script += "printf '%s' '" + errOut + "' >&2\n"
	}
	script += "exit " + itoa(code) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "vitest"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func TestThresholdGate(t *testing.T) {
	t.Run("root thresholds not 100 fail before any exec", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "vitest.config.js"), partialThresholdsJS)
		t.Setenv("PATH", t.TempDir()) // no vitest anywhere: the gate must fail first
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Ok || res.FilesChecked != 1 || res.Errors[0] != thresholdsNot100 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})

	t.Run("fitness-runner root thresholds not 100 fail second", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "vitest.config.js"), fullThresholdsJS)
		// A fake @mayjournal/fitness-shared install makes FitnessRunnerRoot
		// resolve to its config dir, whose thresholds sit at 90.
		pkgDir := filepath.Join(root, "node_modules", "@mayjournal", "fitness-shared")
		writeFile(t, filepath.Join(pkgDir, "package.json"),
			`{"exports":{"./cspell":"./config/cspell.json"}}`)
		writeFile(t, filepath.Join(pkgDir, "config", "cspell.json"), "{}")
		writeFile(t, filepath.Join(pkgDir, "config", "vitest.config.mjs"), partialThresholdsJS)
		t.Setenv("PATH", t.TempDir())
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Ok || res.FilesChecked != 1 || res.Errors[0] != fitnessRunnerThresholdsNot100 {
			t.Fatalf("unexpected result: %+v", res)
		}
	})
}

func TestMissingVitest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "vitest.config.js"), fullThresholdsJS)
	t.Setenv("PATH", t.TempDir())
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 1 || res.Errors[0] != missingVitestHint {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestCoveragePhase(t *testing.T) {
	cases := []struct {
		name      string
		out       string
		errOut    string
		code      int
		ok        bool
		wantError string
	}{
		{"exit 0 passes", "some output", "", 0, true, ""},
		{"stderr last line reported", "", `first line
Coverage for branches (90%) does not meet threshold (100%).`, 1, false,
			"Coverage for branches (90%) does not meet threshold (100%)."},
		{"stderr follows stdout in combined output", "out-line", "err-line", 1, false, "err-line"},
		{"stdout-only failure uses its last line", `line1
line2`, "", 1, false, "line2"},
		{"empty output falls back to run hint", "", "", 1, false, fallbackRunHint},
		{"non-1 exit code still fails with last line", "boom", "", 2, false, "boom"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "vitest.config.js"), fullThresholdsJS)
			fakeBin := t.TempDir()
			installFakeVitest(t, fakeBin, tc.out, tc.errOut, tc.code, "")
			t.Setenv("PATH", fakeBin)
			res, err := run(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 1 {
				t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
			}
			if !tc.ok && res.Errors[0] != tc.wantError {
				t.Fatalf("error = %q, want %q", res.Errors[0], tc.wantError)
			}
		})
	}
}

func TestConfigFallbackArgs(t *testing.T) {
	t.Run("no local config appends --config from fitness-runner root", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "package.json"), fullThresholdsPkgJSON)
		fakeBin := t.TempDir()
		argsFile := filepath.Join(t.TempDir(), "args.txt")
		installFakeVitest(t, fakeBin, "", "", 0, argsFile)
		t.Setenv("PATH", fakeBin)
		res, err := run(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Ok {
			t.Fatalf("unexpected result: %+v", res)
		}
		recorded, err := os.ReadFile(argsFile)
		if err != nil {
			t.Fatal(err)
		}
		// FitnessRunnerRoot falls back to root itself in a bare temp dir.
		want := "run\n--coverage\n--config\n" + filepath.Join(root, fitnessVitestConfig) + "\n"
		if string(recorded) != want {
			t.Fatalf("args = %q, want %q", recorded, want)
		}
	})

	t.Run("local config runs without --config", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "vitest.config.mts"), fullThresholdsJS)
		fakeBin := t.TempDir()
		argsFile := filepath.Join(t.TempDir(), "args.txt")
		installFakeVitest(t, fakeBin, "", "", 0, argsFile)
		t.Setenv("PATH", fakeBin)
		if res, err := run(root, nil); err != nil || !res.Ok {
			t.Fatalf("unexpected result: %+v (%v)", res, err)
		}
		recorded, err := os.ReadFile(argsFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(recorded) != "run\n--coverage\n" {
			t.Fatalf("args = %q, want run/--coverage only", recorded)
		}
	})
}

func TestResolveVitestPrefersNodeModulesBin(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "nested", "repo")
	writeFile(t, filepath.Join(root, "vitest.config.js"), fullThresholdsJS)
	// The walking-up resolution must pick the ancestor's node_modules/.bin
	// over a PATH candidate that would fail the run.
	binDir := filepath.Join(parent, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installFakeVitest(t, binDir, "", "", 0, "")
	pathDir := t.TempDir()
	installFakeVitest(t, pathDir, "", "from PATH, not node_modules", 1, "")
	t.Setenv("PATH", pathDir)
	res, err := run(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected node_modules/.bin vitest (exit 0) to win, got %+v", res)
	}
}

func TestFailureMessage(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
	}{
		{"empty output", "", fallbackRunHint},
		{"whitespace only", " \n \n", fallbackRunHint},
		{"single line", "only line", "only line"},
		{"last of many trimmed", "a\n  b  \n", "b"},
		{"carriage return trimmed", "a\nlast\r\n", "last"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := failureMessage(tc.output); got != tc.want {
				t.Fatalf("failureMessage(%q) = %q, want %q", tc.output, got, tc.want)
			}
		})
	}
}

func TestCombineStreams(t *testing.T) {
	cases := []struct {
		name, stdout, stderr, want string
	}{
		{"both present join with newline", "out", "err", "out\nerr"},
		{"stdout only", "out", "", "out"},
		{"stderr only", "", "err", "err"},
		{"both empty", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := combineStreams(tc.stdout, tc.stderr); got != tc.want {
				t.Fatalf("combineStreams = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestVitestArgsConfigPrecedence(t *testing.T) {
	root := t.TempDir()
	frRoot := t.TempDir()
	got := vitestArgs(root, frRoot)
	want := []string{"run", "--coverage", "--config", filepath.Join(frRoot, fitnessVitestConfig)}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("vitestArgs = %v, want %v", got, want)
	}
}
