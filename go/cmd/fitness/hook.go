package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// runHook dispatches a `fitness hook <name>` invocation. The installed hooks
// are one-line shims that call this, so the orchestration lives here in Go
// rather than in each repo's Bash.
func runHook(name string, args []string) int {
	if err := chdirToplevel(); err != nil {
		fmt.Fprintln(os.Stderr, "fitness hook: "+err.Error())
		return 1
	}
	switch name {
	case "commit-msg":
		return hookCommitMsg(args)
	case "pre-commit":
		return hookPreCommit()
	case "pre-push":
		return hookPrePush()
	default:
		fmt.Fprintln(os.Stderr, "fitness hook: unknown hook "+name)
		return 1
	}
}

// chdirToplevel moves to the git working-tree root so checks and paths resolve
// the same way from any hook's working directory.
func chdirToplevel() error {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}
	return os.Chdir(strings.TrimSpace(string(out)))
}

// hookCommitMsg validates the proposed commit message with semantic-commit,
// plan-trailer, and issue-link-once — the message file is the first argument.
func hookCommitMsg(args []string) int {
	msg, ok := readMessage(args)
	if !ok {
		return 1
	}
	for _, check := range []string{"semantic-commit", "plan-trailer"} {
		if code := run([]string{check, "--message=" + msg}); code != 0 {
			return code
		}
	}
	if amending() {
		_ = os.Setenv("FITNESS_COMMIT_AMEND", "1")
	}
	return run([]string{"issue-link-once", "--message=" + msg})
}

// readMessage reads the commit message file named by the hook's first
// argument, reporting any failure on stderr.
func readMessage(args []string) (string, bool) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "fitness hook commit-msg: missing message file")
		return "", false
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", false
	}
	return string(data), true
}

// amending reports whether the git process running this hook was invoked with
// --amend. The shims exec fitness, so the parent process is git itself.
func amending() bool {
	out, err := exec.Command("ps", "-o", "args=", "-p", strconv.Itoa(os.Getppid())).Output()
	return err == nil && amendArgs(string(out))
}

// amendArgs is the pure test over a git command line.
func amendArgs(args string) bool {
	return slices.Contains(strings.Fields(args), "--amend")
}

// hookPreCommit stamps a staged CHANGELOG entry, runs the full suite, then the
// repo's own build and tests via an optional .githooks/pre-commit.local.
func hookPreCommit() int {
	_ = exec.Command("fitness-stamp-changelog").Run()
	if code := run(nil); code != 0 {
		return code
	}
	return runLocalHook()
}

// runLocalHook runs .githooks/pre-commit.local when it is present and
// executable, so a repo adds its own build without forking the shared hook.
func runLocalHook() int {
	const path = ".githooks/pre-commit.local"
	info, err := os.Stat(path)
	if err != nil || info.Mode()&0o100 == 0 {
		return 0
	}
	cmd := exec.Command(path)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return 1
	}
	return 0
}
