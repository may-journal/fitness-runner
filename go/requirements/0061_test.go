package requirements

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// fiRun runs the candidate `fitness-install <args>` in dir with env added,
// as a user types it, and returns what it printed and its exit code.
func fiRun(t *testing.T, dir string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(installer(t), args...)
	cmd.Dir, cmd.Env = dir, append(userEnv(), env...)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("running fitness-install: %v", err)
	}
	return ansi.ReplaceAllString(string(out), ""), cmd.ProcessState.ExitCode()
}

// fiLastLine is the last printed line, where install-only names its folder.
func fiLastLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// fiFile returns the text of a file the installer wrote.
func fiFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	mustDo(t, err)
	return string(data)
}

// fiVersion matches the release line the installer prints before installing.
var fiVersion = regexp.MustCompile(`fitness: version=(\S+) `)

func Test0061_1(t *testing.T) {
	t.Parallel()
	out, code := fiRun(t, t.TempDir(), nil, "--help")
	sees(t, out, code, 0, "Usage of fitness-install", "-github-action", "-install-hook", "-install-only", "-version")
}

func Test0061_2(t *testing.T) {
	t.Parallel()
	cases := map[string][]string{
		"flag provided but not defined: -unknown":                            {"--unknown"},
		"install-only does not accept runner arguments":                      {"--install-only", "--", "--all"},
		"install-hook cannot be combined with install-only or github-action": {"--install-hook", "--install-only"},
		`invalid release version "invalid"`:                                  {"--version", "invalid"},
	}
	for want, args := range cases {
		out, code := fiRun(t, t.TempDir(), nil, args...)
		sees(t, out, code, 1, "fitness: "+want)
	}
}

func Test0061_3(t *testing.T) {
	t.Parallel()
	first, _ := fiRun(t, t.TempDir(), nil, "--install-only")
	match := fiVersion.FindStringSubmatch(first)
	if match == nil {
		t.Fatalf("no release version in:\n%s", first)
	}
	out, code := fiRun(t, t.TempDir(), nil, "--version", "v"+match[1], "--install-only")
	sees(t, out, code, 0, "fitness: version="+match[1]+" ")
	for _, name := range []string{"fitness", "fitness-check-gofmt", "fitness-check-prose-budget"} {
		if info, err := os.Stat(filepath.Join(fiLastLine(out), name)); err != nil || info.Mode()&0o111 == 0 {
			t.Errorf("printed folder has no runnable %s: %v\n%s", name, err, out)
		}
	}
}

// fiPrivateCache makes a cache holding only the verified release archive,
// copied from the cache the environment already filled, so nothing downloads.
func fiPrivateCache(t *testing.T) []string {
	t.Helper()
	out, code := fiRun(t, t.TempDir(), nil, "--install-only")
	if code != 0 {
		t.Fatalf("installing fitness:\n%s", out)
	}
	key := filepath.Dir(fiLastLine(out))
	cache := t.TempDir()
	copied := filepath.Join(cache, filepath.Base(key))
	mustDo(t, os.MkdirAll(copied, 0o700))
	mustDo(t, os.WriteFile(filepath.Join(copied, "archive.tar.gz"), []byte(fiFile(t, filepath.Join(key, "archive.tar.gz"))), 0o600))
	return []string{"FITNESS_CACHE_DIR=" + cache}
}

func Test0061_4(t *testing.T) {
	t.Parallel()
	env := fiPrivateCache(t)
	first, _ := fiRun(t, t.TempDir(), env, "--install-only")
	again, _ := fiRun(t, t.TempDir(), env, "--install-only")
	if fiLastLine(first) != fiLastLine(again) {
		t.Fatalf("an intact cache was not reused: %q then %q", fiLastLine(first), fiLastLine(again))
	}
	write(t, fiLastLine(first), map[string]string{"fitness-check-markdown-filename-kebab-case": "damaged\n"})
	out, code := fitness(t, example(t, "happyRepo", nil), env, "markdown-filename-kebab-case")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0061_5(t *testing.T) {
	t.Parallel()
	out, code := fiRun(t, t.TempDir(), nil, "--install-hook")
	sees(t, out, code, 1, "fitness: find Git hook directory")
}

func Test0061_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"main.go": "package main\nfunc main(){\n}\n"})
	out, code := fitness(t, repo, nil, "gofmt")
	sees(t, out, code, 1, "main.go: not gofmt-formatted (run: gofmt -w main.go)")
}

// fiAction is a GitHub job's environment, writing its files under dir.
func fiAction(dir string, inputs ...string) []string {
	return append([]string{
		"GITHUB_ACTIONS=true",
		"GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary"),
		"GITHUB_OUTPUT=" + filepath.Join(dir, "output"),
		"GITHUB_PATH=" + filepath.Join(dir, "path"),
	}, inputs...)
}

func Test0061_7(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	out, code := fiRun(t, t.TempDir(), fiAction(dir, "FITNESS_INSTALL_ONLY=true"), "--github-action")
	sees(t, out, code, 0)
	bin := fiLastLine(out)
	sees(t, fiFile(t, filepath.Join(dir, "output")), 0, 0, "bin="+bin+"\n")
	sees(t, fiFile(t, filepath.Join(dir, "path")), 0, 0, bin+"\n")
	sees(t, fiFile(t, filepath.Join(dir, "summary")), 0, 0, "✅ fitness-install", "Verified Fitness binaries installed successfully.")
}

func Test0061_8(t *testing.T) {
	t.Parallel()
	cases := map[string][]string{
		`parsing "maybe": invalid syntax`:         {"FITNESS_INSTALL_ONLY=maybe"},
		"choose action check or checks, not both": {"FITNESS_CHECK=gofmt", "FITNESS_CHECKS=gofmt,go-vet"},
	}
	for want, inputs := range cases {
		dir := t.TempDir()
		out, code := fiRun(t, t.TempDir(), fiAction(dir, inputs...), "--github-action")
		sees(t, out, code, 1, "::error::fitness-install: ", want)
		sees(t, fiFile(t, filepath.Join(dir, "summary")), 0, 0, "❌ fitness-install", want)
	}
}
