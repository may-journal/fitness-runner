package requirements

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func Test0001_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", architectedRepo())
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 0, allPassed())
}

func Test0001_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "README.md: disallowed **bold**", passedOf(1)+", 1 failed")
}

func Test0001_17(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", with(goRepo(), map[string]string{"broken_test.go": "package adder\n\nfunc Test0001_2(t *testing.T {\n"}))
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "[requirements] Info:", "missing ',' in parameter list", "go-vet")
	if !strings.Contains(row(out, "requirements"), "info") || !strings.Contains(row(out, "go-vet"), "failed") {
		t.Errorf("requirements must show info while go-vet sets the exit code, got rows %q and %q", row(out, "requirements"), row(out, "go-vet"))
	}
	out, code = describeFails(t, example(t, "happyRepo", nil), "markdown-no-bold-italic")
	sees(t, out, code, 0, "--describe gave no metadata, so the check ran with defaults", "0 of 1 checks passed, 1 info")
	if !strings.Contains(row(out, "markdown-no-bold-italic"), "info") {
		t.Errorf("a check whose describe fails must still run and show info, got row %q", row(out, "markdown-no-bold-italic"))
	}
	out, code = fitness(t, example(t, "happyRepo", architectedRepo()), []string{killedVet(t)})
	sees(t, out, code, 0, "check ended without a result (signal: killed)", passedOf(1)+", 1 info")
	if !strings.Contains(row(out, "go-vet"), "info") {
		t.Errorf("a killed check must show info while the passing checks set exit 0, got row %q", row(out, "go-vet"))
	}
	repo = example(t, "happyRepo", with(architectedRepo(), map[string]string{"README.md": readme + "\nSome **bold** text.\n"}))
	summary := filepath.Join(t.TempDir(), "summary.md")
	out, code = fitness(t, repo, []string{killedVet(t), "GITHUB_ACTIONS=true", "GITHUB_WORKSPACE=" + repo, "GITHUB_STEP_SUMMARY=" + summary})
	sees(t, out, code, 1, passedOf(2)+", 1 failed, 1 info", "::error file=README.md",
		"::notice title=go-vet::check ended without a result (signal: killed)")
	written, _ := os.ReadFile(summary)
	sees(t, string(written), code, 1, "| go-vet | ℹ️ info |", "### ℹ️ Info")
}

// describeFails runs check in repo with a copy of the installed runner, after
// swapping that check for one whose --describe fails and whose run is the
// real one, as a user who replaced one check would. The runner is placed in
// the bundle by hard link or copy, never a symlink, which Linux resolves back
// to the install folder. It returns the output and exit code.
func describeFails(t *testing.T, repo, check string) (string, int) {
	t.Helper()
	bin := "fitness-check-" + check
	installed := installOnly(t)
	bundle := t.TempDir()
	linkTools(bundle, installed, []string{bin, "fitness"})
	placeRunner(t, filepath.Join(installed, "fitness"), filepath.Join(bundle, "fitness"))
	write(t, bundle, map[string]string{bin: "#!/bin/sh\n[ \"$1\" = --describe ] && exit 1\nexec \"" + filepath.Join(installed, bin) + "\" \"$@\"\n"})
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), check)
		cmd.Dir, cmd.Env = repo, userEnv()
		return cmd
	})
}

func Test0001_14(t *testing.T) {
	t.Parallel()
	// prettier declares no budget, so the repo's 0.3s timeoutMs bounds it,
	// while go-test keeps the budget it declares.
	for _, prelude := range []string{"", "trap '' TERM\n"} {
		repo := example(t, "happyRepo", hungTool("300", prelude))
		out, code := fitness(t, repo, nil)
		sees(t, out, code, 1, "Check timed out after 0.3s")
		if !strings.Contains(row(out, "prettier"), "failed") {
			t.Errorf("a timed-out check must report red, got row %q", row(out, "prettier"))
		}
		if strings.Contains(row(out, "go-test"), "failed") {
			t.Errorf("a check with its own budget must keep it, got row %q", row(out, "go-test"))
		}
		childStopped(t, repo)
	}
	out, code := ownBudget(t)
	sees(t, out, code, 1, "Check timed out after 0.3s")
	if !strings.Contains(row(out, "own-budget"), "failed") {
		t.Errorf("a check past the budget it declares must report red, got row %q", row(out, "own-budget"))
	}
	promptsTimeOut(t)
	timeoutNamesLastLine(t)
}

// promptsTimeOut asserts a check whose git or ssh waits for a password times
// out red at its budget, and the waiting tool is stopped.
func promptsTimeOut(t *testing.T) {
	t.Helper()
	for _, call := range []string{"git fetch origin", "ssh git@git.example.invalid true"} {
		promptTimesOut(t, call)
	}
}

// promptTimesOut asserts what promptsTimeOut does for one call.
func promptTimesOut(t *testing.T, call string) {
	t.Helper()
	out, code, pidFile := prompted(t, call)
	sees(t, out, code, 1, "Check timed out after 0.5s")
	if !strings.Contains(row(out, "prompted"), "failed") {
		t.Errorf("a check whose %s waits for a password must time out red, got row %q", call, row(out, "prompted"))
	}
	processStopped(t, pidFile)
}

// prompted runs, under a 0.5s timeoutMs, a local check that runs call with
// git and ssh waiting for a password, as on a machine with no credentials
// saved. It returns the output, the exit code and the file holding the
// waiting tool's process ID.
func prompted(t *testing.T, call string) (string, int, string) {
	t.Helper()
	path, pidFile := promptingTools(t)
	repo := example(t, "happyRepo", map[string]string{
		".fitnessrc.json":               `{"localChecks": "checks", "timeoutMs": 500}` + "\n",
		"checks/fitness-check-prompted": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"prompted\"}'; exit 0; }\n" + call + "\necho '" + passing + "'\n",
	})
	git(t, repo, "remote", "add", "origin", promptedRemote)
	out, code := fitness(t, repo, []string{path}, "--policy=external", "--checks=prompted")
	return out, code, pidFile
}

// ownBudget runs, beside the installed checks, one that declares a 0.3s
// budget in --describe and then hangs, and returns the output and exit code.
func ownBudget(t *testing.T) (string, int) {
	t.Helper()
	installed, bundle := installOnly(t), t.TempDir()
	linkTools(bundle, installed, []string{"fitness"})
	placeRunner(t, filepath.Join(installed, "fitness"), filepath.Join(bundle, "fitness"))
	write(t, bundle, map[string]string{"fitness-check-own-budget": "#!/bin/sh\n" +
		"[ \"$1\" = --describe ] && { echo '{\"name\":\"own-budget\",\"timeoutMs\":300}'; exit 0; }\nexec /bin/sleep 30\n"})
	repo := example(t, "happyRepo", nil)
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), "--policy=external", "--checks=own-budget,markdown-links")
		cmd.Dir, cmd.Env = repo, userEnv()
		return cmd
	})
}

func Test0001_15(t *testing.T) {
	t.Parallel()
	for _, repo := range []string{example(t, "happyRepo", architectedRepo()), generatedTree(t, architectedRepo(), 3000)} {
		path, calls := loggedGit(t)
		out, code := fitness(t, repo, []string{path})
		sees(t, out, code, 0, allPassed())
		if n := occurrences(calls, "ls-files -z --cached"); n != 1 {
			t.Errorf("git must list the tracked files once for the whole run, but listed them %d times", n)
		}
	}
	stalledGitStops(t)
}

// loggedGit returns a PATH setting whose git logs each call's arguments to
// the returned file, one line each, and then runs the real git.
func loggedGit(t *testing.T) (path, calls string) {
	t.Helper()
	real, err := exec.LookPath("git")
	mustDo(t, err)
	dir := t.TempDir()
	calls = filepath.Join(t.TempDir(), "git.log")
	write(t, dir, map[string]string{"git": "#!/bin/sh\necho \"$*\" >> \"" + calls + "\"\nexec \"" + real + "\" \"$@\"\n"})
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"), calls
}

// occurrences counts how often want appears in the file at path.
func occurrences(path, want string) int {
	raw, _ := os.ReadFile(path)
	return strings.Count(string(raw), want)
}

func Test0001_16(t *testing.T) {
	t.Parallel()
	other := example(t, "happyRepo", map[string]string{"other.md": "# Other\n\nSome **bold** text.\n"})
	installed, bundle := installOnly(t), t.TempDir()
	linkTools(bundle, installed, []string{"fitness"})
	placeRunner(t, filepath.Join(installed, "fitness"), filepath.Join(bundle, "fitness"))
	walk := "exec \"" + filepath.Join(installed, "fitness-check-markdown-no-bold-italic") + "\" --root "
	write(t, bundle, map[string]string{
		// Each runs the real check under the list its run handed it: one
		// after staging a new file, one on another repo.
		"fitness-check-staged-later": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\":\"staged-later\"}'; exit 0; }\n" +
			"printf '# New\\n\\nSome **bold** text.\\n' > new.md\ngit add new.md\n" + walk + "\"$PWD\"\n",
		"fitness-check-other-repo": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\":\"other-repo\"}'; exit 0; }\n" + walk + "\"" + other + "\"\n",
	})
	repo := example(t, "happyRepo", nil)
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), "--policy=external", "--checks=staged-later,other-repo")
		cmd.Dir, cmd.Env = repo, userEnv()
		return cmd
	})
	sees(t, out, code, 1, "new.md: disallowed **bold**", "other.md: disallowed **bold**")
}

func Test0001_13(t *testing.T) {
	t.Parallel()
	// slow-zz has the longest budget of any check. It sits after one slow
	// check per pool slot and the installed suite, so catalog order alone
	// would hold it until another check ends.
	budgets := map[string]int{"slow-zz": 1800000}
	for i := range runtime.NumCPU() {
		budgets[fmt.Sprintf("slow-%02d", i)] = 30000
	}
	out, code, stamps := timedRun(t, budgets)
	sees(t, out, code, 0, fmt.Sprintf("All %d checks passed", len(strings.Fields(catalogRows))+len(budgets)))
	describes, runs := stamps.phase(t, "describe", budgets), stamps.phase(t, "run", budgets)
	// One core gives the pool one slot, so only there may steps not overlap.
	if runtime.NumCPU() > 1 && overlapping(describes) < 2 {
		t.Errorf("the describe steps must overlap, but none started before the first ended: %v", describes)
	}
	if long, first := runs["slow-zz"], earliestEnd(runs, "slow-zz"); !long.start.Before(first) {
		t.Errorf("slow-zz has the longest budget, so it must start before any other check ends: it started %v, the first end was %v", long.start, first)
	}
	keepsCatalogOrder(t, out, slices.Sorted(maps.Keys(budgets)))
	showsProgress(t)
}

// keepsCatalogOrder asserts the table lists the slow checks in want's order.
func keepsCatalogOrder(t *testing.T, out string, want []string) {
	t.Helper()
	slow := slices.DeleteFunc(rowCells(out), func(c string) bool { return !strings.HasPrefix(c, "slow-") })
	if !slices.Equal(slow, want) {
		t.Errorf("the table must keep catalog order %v, got %v", want, slow)
	}
}

// timedPrefix names the slow checks Test0001_13 installs. Run under such a
// name, this test binary is that check, and logs when each step starts and
// ends.
const timedPrefix = "fitness-check-slow-"

// timedRun adds one slow check per budget to a copy of the installed suite,
// each this test binary under the check's name, and runs fitness on a
// passing repo. It returns the output, the exit code and the times each
// slow check logged.
func timedRun(t *testing.T, budgets map[string]int) (string, int, stamps) {
	t.Helper()
	installed, bundle, logs := installOnly(t), t.TempDir(), t.TempDir()
	linkTools(bundle, installed, []string{"fitness"})
	placeRunner(t, filepath.Join(installed, "fitness"), filepath.Join(bundle, "fitness"))
	self, err := os.Executable()
	mustDo(t, err)
	for name, ms := range budgets {
		mustDo(t, os.Symlink(self, filepath.Join(bundle, "fitness-check-"+name)))
		write(t, logs, map[string]string{name + ".budget": strconv.Itoa(ms)})
	}
	repo := example(t, "happyRepo", architectedRepo())
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"))
		cmd.Dir, cmd.Env = repo, append(userEnv(), "FITNESS_TIMED="+logs)
		return cmd
	})
	return out, code, stamps(logs)
}

// timedCheck is one slow check: it logs its step's start, pauses, logs the
// end, then answers --describe with its budget or a run with a pass.
func timedCheck(name string, args []string) int {
	check := strings.TrimPrefix(name, "fitness-check-")
	dir := os.Getenv("FITNESS_TIMED")
	step, pause, answer := "run", time.Second, `{"ok":true,"errors":[],"filesChecked":0}`
	if slices.Contains(args, "--describe") {
		budget, err := os.ReadFile(filepath.Join(dir, check+".budget"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "slow check has no budget:", err)
			return 3
		}
		step, pause, answer = "describe", 300*time.Millisecond, `{"name":"`+check+`","timeoutMs":`+string(budget)+`}`
	}
	logCall(filepath.Join(dir, check+".log"), step+" start "+strconv.FormatInt(time.Now().UnixNano(), 10))
	time.Sleep(pause)
	logCall(filepath.Join(dir, check+".log"), step+" end "+strconv.FormatInt(time.Now().UnixNano(), 10))
	fmt.Println(answer)
	return 0
}

// stamps is the folder of the slow checks' logs.
type stamps string

// span is when one step of one check started and ended.
type span struct{ start, end time.Time }

// phase returns each check's span for step, read from its log. A retried
// describe keeps its last try. A check that logged no start or end for
// step fails the test, so no comparison runs on a missing time.
func (s stamps) phase(t *testing.T, step string, checks map[string]int) map[string]span {
	t.Helper()
	spans := map[string]span{}
	for check := range checks {
		raw, _ := os.ReadFile(filepath.Join(string(s), check+".log"))
		got := stepSpan(string(raw), step)
		if got.start.IsZero() || got.end.IsZero() {
			t.Fatalf("%s logged no full %s step: %q", check, step, raw)
		}
		spans[check] = got
	}
	return spans
}

// stepSpan reads one check's log lines, "<step> start|end <unix ns>", into
// the span of step.
func stepSpan(log, step string) span {
	var got span
	for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != step {
			continue
		}
		ns, _ := strconv.ParseInt(fields[2], 10, 64)
		if fields[1] == "start" {
			got.start = time.Unix(0, ns)
		} else {
			got.end = time.Unix(0, ns)
		}
	}
	return got
}

// overlapping counts the spans that start before the first one ends.
func overlapping(spans map[string]span) int {
	first := earliestEnd(spans, "")
	n := 0
	for _, s := range spans {
		if s.start.Before(first) {
			n++
		}
	}
	return n
}

// earliestEnd is when the first span other than skip's ended.
func earliestEnd(spans map[string]span, skip string) time.Time {
	var first time.Time
	for name, s := range spans {
		if name != skip && (first.IsZero() || s.end.Before(first)) {
			first = s.end
		}
	}
	return first
}

// showsProgress asserts the heartbeat names a slow check with its time and
// budget, and that the check, once alone, streams its stderr live.
func showsProgress(t *testing.T) {
	t.Helper()
	out, code := slowCheck(t, 20000, "sleep 7", progressOn)
	sees(t, out, code, 0, "  running: slow (budget 20s)")
	streamedLive(t, out)
}

// timeoutNamesLastLine asserts a timed-out check's row shows its last stderr
// line, and that off a terminal it was named once at half its budget.
func timeoutNamesLastLine(t *testing.T) {
	t.Helper()
	out, code := slowCheck(t, 1000, "exec /bin/sleep 30")
	sees(t, out, code, 1, "Check timed out after 1s", "Last output: "+slowFirst)
	if n := strings.Count(out, "running: slow (budget 1s)"); n != 1 {
		t.Errorf("off a terminal, a check past half its budget must be named once, but was named %d times:\n%s", n, out)
	}
}

// progressOn asks for the heartbeat, as a terminal would.
const progressOn = "FITNESS_PROGRESS=1"

// slowFirst and slowLast are the stderr lines slowCheck's check writes
// before and after it waits.
const slowFirst, slowLast = "compiling 1 of 3", "compiling 3 of 3"

// slowCheck runs a local check, slow, beside a quick one. Slow declares
// budgetMs, writes slowFirst to stderr, runs wait, then writes slowLast and
// passes. It returns the output and exit code.
func slowCheck(t *testing.T, budgetMs int, wait string, env ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{
		".fitnessrc.json": `{"localChecks": "checks"}` + "\n",
		"checks/fitness-check-slow": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"slow\", \"timeoutMs\": " + strconv.Itoa(budgetMs) + "}'; exit 0; }\n" +
			"echo '" + slowFirst + "' >&2\n" + wait + "\necho '" + slowLast + "' >&2\necho '" + passing + "'\n",
		"checks/fitness-check-quick": localCheck("quick", passing)["checks/fitness-check-quick"],
	})
	return fitness(t, repo, env, "--policy=external", "--checks=slow,quick")
}

// streamedLive asserts slow's stderr streamed once, as it ran: its first
// line before the heartbeat named it, its last line after.
func streamedLive(t *testing.T, out string) {
	t.Helper()
	first, beat, last := strings.Index(out, slowFirst), strings.Index(out, "running: slow"), strings.Index(out, slowLast)
	if first < 0 || first > beat || beat > last {
		t.Errorf("slow's stderr must stream as it runs, around the heartbeat:\n%s", out)
	}
	if n := strings.Count(out, slowFirst); n != 1 {
		t.Errorf("streamed stderr must not print again after the run, but printed %d times:\n%s", n, out)
	}
}

func Test0001_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, _ := fitness(t, repo, nil)
	for _, bin := range bundleChecks(t, installOnly(t)) {
		if row(out, strings.TrimPrefix(bin, "fitness-check-")) == "" && bin != "fitness-check-go-test-coverage" {
			t.Errorf("%s is installed but fitness never ran it", bin)
		}
	}
}

// childStopped asserts the hung prettier's child process is gone.
func childStopped(t *testing.T, repo string) {
	t.Helper()
	processStopped(t, filepath.Join(repo, "prettier.pid"))
}

// processStopped asserts the process whose ID pidFile holds is gone.
func processStopped(t *testing.T, pidFile string) {
	t.Helper()
	raw, err := os.ReadFile(pidFile)
	mustDo(t, err)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if !gone(pid, 3*time.Second) {
		t.Fatalf("process %d from %s is still running after fitness stopped it", pid, filepath.Base(pidFile))
	}
}

// readme is happyRepo's README, for Givens that add to it.
const readme = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# App\n\nShort and clean.\n"

// bundleChecks lists the check programs the installer put in dir.
func bundleChecks(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	mustDo(t, err)
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "fitness-check-") {
			names = append(names, e.Name())
		}
	}
	return names
}

// installOnly runs `fitness-install --install-only`, which prints the
// directory holding the runner and every check, and returns that directory.
func installOnly(t *testing.T) string {
	t.Helper()
	out, err := exec.Command(installer(t), "--install-only").Output()
	mustDo(t, err)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// gone polls until pid no longer exists or the deadline passes.
func gone(pid int, within time.Duration) bool {
	for deadline := time.Now().Add(within); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if syscall.Kill(pid, 0) != nil {
			return true
		}
	}
	return false
}

// stalledGitStops asserts a run whose git call listing tracked or staged
// files stalls, under no-prompt mode, ends at the call's ten-second deadline
// as a setup error naming it, and the stalled git is stopped.
func stalledGitStops(t *testing.T) {
	t.Helper()
	for call, named := range map[string]string{
		"ls-files": "git ls-files -z --cached did not finish within 10s",
		"diff":     "git diff --cached --name-only did not finish within 10s",
	} {
		path, pidFile := promptingTools(t, call)
		start := time.Now()
		out, code := fitness(t, example(t, "happyRepo", architectedRepo()), []string{path, "GIT_TERMINAL_PROMPT=0"})
		sees(t, out, code, 1, "fitness setup: "+named)
		if took := time.Since(start); took > 30*time.Second {
			t.Errorf("a stalled git %s must end the run at its deadline, but the run took %s", call, took)
		}
		processStopped(t, pidFile)
	}
}
