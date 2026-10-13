package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/conf"
	"github.com/may-journal/fitness-runner/go/internal/selection"
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
// plan-trailer, requirements, and issue-link-once — the message file is the
// first argument. Requirements pairs tests with docs here, where an
// exemption trailer is readable.
func hookCommitMsg(args []string) int {
	msg, ok := readMessage(args)
	if !ok {
		return 1
	}
	if amending() {
		_ = os.Setenv("FITNESS_COMMIT_AMEND", "1")
	}
	for _, check := range commitMsgChecks() {
		if code := run([]string{check, "--message=" + msg}); code != 0 {
			return code
		}
	}
	return 0
}

// commitMsgChecks lists the checks the commit-msg hook runs. Requirements
// runs only when the repo's suite does, so an external policy that leaves
// it out never meets it at commit time. A check external disabledChecks
// turns off never runs.
func commitMsgChecks() []string {
	checks := []string{"semantic-commit", "plan-trailer", "requirements", "issue-link-once"}
	cfg, err := loadConfig(".")
	if err != nil {
		return checks
	}
	return slices.DeleteFunc(checks, func(c string) bool {
		return turnedOff(cfg, c) || c == "requirements" && leavesOut(cfg, c)
	})
}

// turnedOff reports whether an external policy's disabledChecks names name,
// so a hook skips it rather than fail on an empty selection.
func turnedOff(cfg *conf.Config, name string) bool {
	return cfg != nil && cfg.Policy == selection.External && slices.Contains(cfg.DisabledChecks, name)
}

// leavesOut reports whether an external policy's check list omits name.
func leavesOut(cfg *conf.Config, name string) bool {
	if cfg == nil || cfg.Policy != selection.External {
		return false
	}
	return len(cfg.LegacyChecks) > 0 && !slices.Contains(cfg.LegacyChecks, name)
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

// hookPreCommit runs file checks, then the repo's own build and tests via an
// optional .githooks/pre-commit.local.
func hookPreCommit() int {
	_ = os.Setenv("FITNESS_INTERNAL_HOOK", "pre-commit")
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
