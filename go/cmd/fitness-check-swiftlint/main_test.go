package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const passOutput = "[\n\n]\n"

const failOutput = `[{"character":9,"file":"/repo/Bad.swift","line":4,` +
	`"reason":"Variable name 'x' should be between 3 and 40 characters long",` +
	`"rule_id":"identifier_name","severity":"Error","type":"Identifier Name"}]`

const failFormatted = "/repo/Bad.swift:4:9 - Variable name 'x' should be " +
	"between 3 and 40 characters long (identifier_name)"

// fakeSwiftlint installs a scripted swiftlint on PATH that records its argv
// and working directory, prints output verbatim, and exits with exitCode.
// It returns the path of the recording file.
func fakeSwiftlint(t *testing.T, output string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	outFile := filepath.Join(dir, "output.txt")
	if err := os.WriteFile(outFile, []byte(output), 0o644); err != nil {
		t.Fatal(err)
	}
	callFile := filepath.Join(dir, "call.txt")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + callFile + "\"\n" +
		"pwd -P >> \"" + callFile + "\"\n" +
		"cat \"" + outFile + "\"\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "swiftlint"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	return callFile
}

func TestRunJudgment(t *testing.T) {
	cases := []struct {
		name       string
		output     string
		exitCode   int
		ok         bool
		wantErrors []string
	}{
		{"passes on an empty violation array", passOutput, 0, true, nil},
		{"fails and formats violations from the JSON reporter", failOutput, 2,
			false, []string{failFormatted}},
		{"reports NotInstalled on command-not-found output",
			"/bin/sh: swiftlint: command not found\n", 127,
			false, []string{"SwiftLint not installed: brew install swiftlint"}},
		{"skips clean when there are no Swift files",
			"Error: No lintable files found at paths: '.'\n", 1, true, nil},
		{"falls back to the run hint on malformed output",
			"swiftlint crashed unexpectedly", 1,
			false, []string{"swiftlint reported an error (run: swiftlint lint --strict)"}},
		{"fails on violations even at exit 0", failOutput, 0,
			false, []string{failFormatted}},
		{"omits the column when character is null",
			`[{"character":null,"file":"/repo/Bad.swift","line":4,` +
				`"reason":"Files should end with a newline","rule_id":"trailing_newline"}]`, 2,
			false, []string{"/repo/Bad.swift:4 - Files should end with a newline (trailing_newline)"}},
		{"non-array JSON parses to no violations and hits the run hint",
			`{"reason":"not an array"}`, 2,
			false, []string{"swiftlint reported an error (run: swiftlint lint --strict)"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeSwiftlint(t, tc.output, tc.exitCode)
			res, err := run(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 0 {
				t.Fatalf("filesChecked = %d, want 0", res.FilesChecked)
			}
			if len(res.Errors) != len(tc.wantErrors) {
				t.Fatalf("errors = %v, want %v", res.Errors, tc.wantErrors)
			}
			for i, want := range tc.wantErrors {
				if res.Errors[i] != want {
					t.Fatalf("error[%d] = %q, want %q", i, res.Errors[i], want)
				}
			}
		})
	}
}

func TestRunInvocation(t *testing.T) {
	callFile := fakeSwiftlint(t, passOutput, 0)
	root := t.TempDir()
	if _, err := run(root, nil); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(callFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(recorded), "\n"), "\n")
	wantArgs := []string{"lint", "--strict", "--reporter", "json", "--quiet", "."}
	if len(lines) != len(wantArgs)+1 {
		t.Fatalf("recorded lines = %v, want %d args + cwd", lines, len(wantArgs))
	}
	for i, want := range wantArgs {
		if lines[i] != want {
			t.Fatalf("arg[%d] = %q, want %q", i, lines[i], want)
		}
	}
	wantCwd, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if lines[len(lines)-1] != wantCwd {
		t.Fatalf("cwd = %q, want %q", lines[len(lines)-1], wantCwd)
	}
}

func TestRunMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.FilesChecked != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.Errors) != 1 || res.Errors[0] != "SwiftLint not installed: brew install swiftlint" {
		t.Fatalf("errors = %v, want the install hint", res.Errors)
	}
}
