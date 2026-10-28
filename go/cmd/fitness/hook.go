package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// runHook dispatches a `fitness hook <name>` invocation. The installed hooks
// are one-line shims that call this, so the orchestration lives here in Go
// rather than in each repo's Bash.
func runHook(name string, args []string) int {
	if err := chdirToplevel(); err != nil {
		fmt.Fprintln(os.Stderr, "fitness hook: "+err.Error())
		return 1
	}
	repairHooksPath()
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
// plan-trailer, and issue-link-once — the message file is the first
// argument. It also runs requirements-changed on the staged files. An amend
// is judged against the amended commit's parent. The four run in one pool, sharing one table, and any failure fails
// the hook.
func hookCommitMsg(args []string) int {
	msg, ok := readMessage(args)
	if !ok {
		return 1
	}
	markAmend()
	return runMessageChecks(commitMsgChecks(), msg)
}

// runMessageChecks resolves each named check as a single-check run would,
// then runs those that read the message from the environment at once, and
// any others one by one as before. No names run nothing.
func runMessageChecks(names []string, msg string) int {
	if len(names) == 0 {
		return 0
	}
	root, checks, err := resolveMessageChecks(names)
	if err != nil {
		report.Error("fitness setup", err)
		return 1
	}
	pooled, alone := splitByMessageArg(checks)
	code := 0
	if len(pooled) > 0 {
		env := append(contextEnv(root, pooled, false, nil), "FITNESS_CTX_MESSAGE="+msg)
		code = runChecks(root, "", pooled, nil, env, func() {})
	}
	for _, check := range alone {
		code = max(code, run([]string{check.name, "--message=" + msg}))
	}
	return code
}

// splitByMessageArg separates the checks that read the message from
// FITNESS_CTX_MESSAGE, which the runner sets for any check naming a
// context argument, from those, such as a repo's own override, that take
// --message on their command line and so run one at a time.
func splitByMessageArg(checks []resolved) (pooled, alone []resolved) {
	for _, check := range checks {
		if check.desc.ContextInlineArg != "" {
			pooled = append(pooled, check)
		} else {
			alone = append(alone, check)
		}
	}
	return pooled, alone
}

// resolveMessageChecks finds the repository root and resolves names there,
// failing first when the install is broken.
func resolveMessageChecks(names []string) (string, []resolved, error) {
	root, err := os.Getwd()
	if err == nil {
		err = installErr
	}
	if err != nil {
		return "", nil, err
	}
	checks, err := resolveEach(root, names)
	return root, checks, err
}

// resolveEach resolves every name as `fitness <name>` would, in order.
func resolveEach(root string, names []string) ([]resolved, error) {
	var checks []resolved
	for _, name := range names {
		named, _, err := selectedChecks(root, []string{name})
		if err != nil {
			return nil, err
		}
		checks = append(checks, named...)
	}
	return checks, nil
}

// commitMsgChecks lists the checks the commit-msg hook runs, without any
// that an external policy's disabledChecks turns off.
func commitMsgChecks() []string {
	checks := []string{"semantic-commit", "plan-trailer", "issue-link-once", "requirements-changed"}
	cfg, err := loadConfig(".")
	if err != nil {
		return checks
	}
	return slices.DeleteFunc(checks, func(c string) bool { return turnedOff(cfg, c) })
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

// markAmend tells the checks when the commit amends HEAD.
func markAmend() {
	if amending() {
		_ = os.Setenv("FITNESS_COMMIT_AMEND", "1")
	}
}

// amending reports whether the git process running this hook was invoked with
// --amend. The shims exec the installer, which execs fitness, so the parent process is git itself.
func amending() bool {
	out, err := exec.Command("ps", "-o", "args=", "-p", strconv.Itoa(os.Getppid())).Output()
	return err == nil && amendArgs(string(out))
}

// amendArgs is the pure test over a git command line.
func amendArgs(args string) bool {
	return slices.Contains(strings.Fields(args), "--amend")
}

// hookPreCommit runs file checks, then the repo's own build and tests via an
// optional .githooks/pre-commit.local. The requirements check judges an
// amend on everything it changes since the amended commit's parent.
func hookPreCommit() int {
	_ = os.Setenv("FITNESS_INTERNAL_HOOK", "pre-commit")
	markAmend()
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
