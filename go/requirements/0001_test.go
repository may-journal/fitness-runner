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

func Test0001_18(t *testing.T) {
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
	promptsRefused(t)
	offlineCommitAndPush(t)
}

// offlineWithin bounds a commit or a push behind a proxy that never answers:
// the runner's two-second probe plus the hooks around it.
const offlineWithin = 10 * time.Second

// offlineCommitAndPush commits and pushes in a JavaScript repo with a GitHub
// origin behind a silent proxy, each step within offlineWithin.
func offlineCommitAndPush(t *testing.T) {
	t.Helper()
	repo, env := hooked(t, map[string]string{"package.json": `{"dependencies": {"left-pad": "^1.0.0"}}` + "\n",
		".fitnessrc.json": `{"policy": "external", "checks": ["dependency-currency", "local-branch"]}` + "\n"})
	recordedNow(t, env)
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "add", "origin", "https://github.com/may-journal/fitness-runner.git")
	git(t, repo, "remote", "set-url", "--push", "origin", bare)
	proxy := silentListener(t)
	env = append(env, "HTTPS_PROXY="+proxy, "NO_PROXY=", "no_proxy=", "GITHUB_API_URL=https://api.github.com", "GH_HOST=github.com")
	note := "api.github.com did not answer within 2s through proxy " + proxy + ", so fitness skipped "
	steps := [][]string{{"commit", "--allow-empty", "-m", "feat(app): add a feature"}, {"push", "origin", "HEAD:main"}}
	wants := [][]string{{note + "the npm registry reads"}, {"pre-push: info: " + note + "the Plan gate", note + "the PR and Plan reads"}}
	for i, step := range steps {
		start := time.Now()
		out, code := user(t, repo, env, step...)
		sees(t, out, code, 0, wants[i]...)
		if took := time.Since(start); took > offlineWithin {
			t.Errorf("git %s took %s behind a silent proxy, want under %s", step[0], took.Round(time.Second), offlineWithin)
		}
	}
}

// recordedNow writes a latest release lookup of the candidate's version,
// made just now, into the private cache env names.
func recordedNow(t *testing.T, env []string) {
	t.Helper()
	i := slices.IndexFunc(env, func(kv string) bool { return strings.HasPrefix(kv, "FITNESS_CACHE_DIR=") })
	if i < 0 {
		t.Fatalf("no private cache in %q", env)
	}
	record := fmt.Sprintf(`{"tag":"v%s","checked":%q}`, rlCandidate(t), time.Now().Format(time.RFC3339))
	mustDo(t, os.WriteFile(filepath.Join(strings.TrimPrefix(env[i], "FITNESS_CACHE_DIR="), "latest-release.json"), []byte(record), 0o600))
}

// describeFails runs check in repo after swapping it for one whose --describe
// fails, linking or copying the runner because Linux resolves symlinks back.
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
	// A 0.3s fast total stops prettier, the one fast check, so it starts on
	// a loaded machine. That total never stops go-test, in the test stage.
	for _, prelude := range []string{"", "trap '' TERM\n"} {
		repo := example(t, "happyRepo", hungTool(prelude))
		out, code := fitness(t, repo, []string{fastTotal(300)}, "--policy=external", "--checks=prettier,go-test")
		sees(t, out, code, 1, "Stopped at the 0.3s fast total")
		if !strings.Contains(row(out, "prettier"), "failed") {
			t.Errorf("a stopped check must report red, got row %q", row(out, "prettier"))
		}
		if strings.Contains(row(out, "go-test"), "failed") {
			t.Errorf("the fast total must never stop a test-stage check, got row %q", row(out, "go-test"))
		}
		childStopped(t, repo)
	}
	promptsTimeOut(t)
	timeoutNamesLastLine(t)
	hugeTreeHints(t)
	timesTrees(t)
}

// promptsTimeOut asserts a check that turns git prompts back on stops red at
// the fast total, and its waiting tool is stopped.
func promptsTimeOut(t *testing.T) {
	t.Helper()
	for _, call := range []string{"GIT_TERMINAL_PROMPT=1 git fetch origin", "ssh git@git.example.invalid true"} {
		promptTimesOut(t, call)
	}
}

// promptTimesOut asserts what promptsTimeOut does for one call.
func promptTimesOut(t *testing.T, call string) {
	t.Helper()
	out, code, pidFile := prompted(t, call)
	sees(t, out, code, 1, "Stopped at the 0.5s fast total")
	if !strings.Contains(row(out, "prompted"), "failed") {
		t.Errorf("a check whose %s waits for a password must time out red, got row %q", call, row(out, "prompted"))
	}
	processStopped(t, pidFile)
}

// prompted runs a local check whose git and ssh await a password, under a
// 0.5s fast total. It returns output, exit code and process-ID file.
func prompted(t *testing.T, call string, env ...string) (string, int, string) {
	t.Helper()
	path, pidFile := promptingTools(t)
	repo := example(t, "happyRepo", map[string]string{
		".fitnessrc.json":               `{"localChecks": "checks"}` + "\n",
		"checks/fitness-check-prompted": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"prompted\"}'; exit 0; }\n" + call + "\necho '" + passing + "'\n",
	})
	git(t, repo, "remote", "add", "origin", promptedRemote)
	out, code := fitness(t, repo, append([]string{path, fastTotal(500)}, env...), "--policy=external", "--checks=prompted")
	return out, code, pidFile
}

// promptsRefused asserts a check whose git, over HTTPS or ssh, meets a
// password prompt ends in time as info naming the remote.
func promptsRefused(t *testing.T) {
	t.Helper()
	for remote, call := range map[string]string{promptedRemote: "git fetch origin", "git@git.example.invalid": "git fetch " + sshRemote} {
		out, code, _ := prompted(t, call+" || exit 1")
		sees(t, out, code, 0, remote+" asked for a password, and checks never wait for one", "1 info")
		if !strings.Contains(row(out, "prompted"), "info") {
			t.Errorf("a check whose %s may not ask for a password must show info, got row %q", call, row(out, "prompted"))
		}
	}
}

func Test0001_15(t *testing.T) {
	t.Parallel()
	for _, repo := range []string{linkedRepo(t), generatedTree(t, architectedRepo(), 3000)} {
		path, calls := loggedGit(t)
		out, code := fitness(t, repo, []string{path})
		sees(t, out, code, 0, allPassed())
		if n := occurrences(calls, "ls-files -z --stage"); n != 1 {
			t.Errorf("git must list the tracked files once, and the runner stat them once, for the whole run, but listed them %d times", n)
		}
		if n := occurrences(calls, "ls-files -z --cached"); n != 0 {
			t.Errorf("every check must use the shared list, but %d listed their own", n)
		}
	}
	hugeProductCommit(t)
	stalledGitStops(t)
}

// linkedRepo is architectedRepo with a tracked symlink to an untracked
// doc holding bold text, and a tracked doc deleted from disk. The shared
// list drops both, so no check reads the bold text or misses the doc.
func linkedRepo(t *testing.T) string {
	t.Helper()
	repo := example(t, "happyRepo", with(architectedRepo(), map[string]string{"docs/gone.md": "# Gone\n\nA doc.\n"}))
	mustDo(t, os.Symlink("../notes.txt", filepath.Join(repo, "docs", "linked.md")))
	git(t, repo, "add", "docs/linked.md")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "docs(linked): link the notes")
	write(t, repo, map[string]string{"notes.txt": "# Notes\n\nSome **bold** text.\n", "docs/gone.md": ""})
	return repo
}

// hugeProductCommit asserts that in a repo of 500,000 tracked files a one-file
// commit passes every architecture check inside its budget. No check may diff the whole work tree.
func hugeProductCommit(t *testing.T) {
	t.Helper()
	repo := generatedTree(t, architectedRepo(), 500000)
	defer os.RemoveAll(repo)
	got, code := fitness(t, repo, installerEnv(t), "init")
	sees(t, got, code, 0, "fitness init: installed hooks")
	stagedOnlyDiffs(t, repo)
	out, code, _ := productCommit(t, repo)
	sees(t, out, code, 0)
	for _, check := range architectureChecks {
		if !strings.Contains(row(out, check), "│ passed") {
			t.Errorf("a product commit in a 500,000-file repo must pass %s in its budget, got row %q", check, row(out, check))
		}
	}
}

// stagedOnlyDiffs stages productEdit in repo and runs `fitness hook pre-commit`
// through a git that logs each call. Git puts its own folder first on a hook's
// PATH, so only a direct run reaches the logger.
func stagedOnlyDiffs(t *testing.T, repo string) {
	t.Helper()
	edit := productEdit()
	write(t, repo, edit)
	git(t, repo, append([]string{"add"}, slices.Collect(maps.Keys(edit))...)...)
	path, calls := loggedGit(t)
	out, code := fitness(t, repo, []string{path}, "hook", "pre-commit")
	sees(t, out, code, 0)
	if n := wholeTreeDiffs(calls); n != 0 {
		t.Errorf("a staged product edit must diff only the files it changed, but %d diffs read all 500,000 files", n)
	}
}

// wholeTreeDiffs counts the logged `git diff -U0` calls that name no path.
func wholeTreeDiffs(calls string) int {
	raw, _ := os.ReadFile(calls)
	n := 0
	for _, call := range strings.Split(string(raw), "\n") {
		if strings.Contains(call, "diff -U0") && !strings.Contains(call, " -- ") {
			n++
		}
	}
	return n
}

// loggedGit returns a PATH setting whose git logs each call's arguments to the
// returned file, one line each, before running the real git.
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
	// slow-zz is a test runner but sits last in catalog order, so it would
	// wait for a free pool slot.
	stages := map[string]string{"slow-zz": "tests"}
	for i := range runtime.NumCPU() {
		stages[fmt.Sprintf("slow-%02d", i)] = "fast"
	}
	out, code, stamps := timedRun(t, stages)
	sees(t, out, code, 0, fmt.Sprintf("All %d checks passed", len(strings.Fields(catalogRows))+len(stages)))
	describes, runs := stamps.phase(t, "describe", stages), stamps.phase(t, "run", stages)
	// One core gives the pool one slot, so only there may steps not overlap.
	if runtime.NumCPU() > 1 && overlapping(describes) < 2 {
		t.Errorf("the describe steps must overlap, but none started before the first ended: %v", describes)
	}
	if long, first := runs["slow-zz"], earliestEnd(runs, "slow-zz"); !long.start.Before(first) {
		t.Errorf("slow-zz is a test runner, so it must start before any other check ends: it started %v, the first end was %v", long.start, first)
	}
	keepsCatalogOrder(t, out, slices.Sorted(maps.Keys(stages)))
	showsProgress(t)
	runnerTakesNoSlot(t)
}

// runnerTakesNoSlot asserts a test runner takes no pool slot. With one fast
// check per core, all of them start before any ends.
func runnerTakesNoSlot(t *testing.T) {
	t.Helper()
	stages := map[string]string{"slow-zz": "tests"}
	for i := range runtime.NumCPU() {
		stages[fmt.Sprintf("slow-%02d", i)] = "fast"
	}
	_, _, stamps := timedRun(t, stages, "--policy=external", "--checks="+strings.Join(slices.Sorted(maps.Keys(stages)), ","))
	runs := stamps.phase(t, "run", stages)
	if n := overlapping(runs); n != len(runs) {
		t.Errorf("a test runner must take no pool slot, so all %d checks start before the first ends, but %d did: %v", len(runs), n, runs)
	}
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

// timedRun adds one slow check per stage entry to the installed suite. It runs
// fitness with args on a passing repo and returns the logged times.
func timedRun(t *testing.T, stages map[string]string, args ...string) (string, int, stamps) {
	t.Helper()
	installed, bundle, logs := installOnly(t), t.TempDir(), t.TempDir()
	linkTools(bundle, installed, []string{"fitness"})
	placeRunner(t, filepath.Join(installed, "fitness"), filepath.Join(bundle, "fitness"))
	self, err := os.Executable()
	mustDo(t, err)
	for name, stage := range stages {
		mustDo(t, os.Symlink(self, filepath.Join(bundle, "fitness-check-"+name)))
		write(t, logs, map[string]string{name + ".stage": stage})
	}
	repo := example(t, "happyRepo", architectedRepo())
	out, code := execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), args...)
		cmd.Dir, cmd.Env = repo, append(userEnv(), "FITNESS_TIMED="+logs)
		return cmd
	})
	return out, code, stamps(logs)
}

// timedCheck is one slow check that logs its step's start and end around a
// pause, answering --describe with its stage.
func timedCheck(name string, args []string) int {
	check := strings.TrimPrefix(name, "fitness-check-")
	dir := os.Getenv("FITNESS_TIMED")
	step, pause, answer := "run", time.Second, `{"ok":true,"errors":[],"filesChecked":0}`
	if slices.Contains(args, "--describe") {
		stage, err := os.ReadFile(filepath.Join(dir, check+".stage"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "slow check has no stage:", err)
			return 3
		}
		step, pause, answer = "describe", 300*time.Millisecond, `{"name":"`+check+`","stage":"`+string(stage)+`"}`
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

// phase returns each check's span for step from its log, keeping a retried
// describe's last try. A check that logged no start or end for step fails the
// test, so no comparison runs on a missing time.
func (s stamps) phase(t *testing.T, step string, checks map[string]string) map[string]span {
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
// limit. Once alone, the check streams its stderr live.
func showsProgress(t *testing.T) {
	t.Helper()
	out, code := slowCheck(t, "sleep 7", progressOn, fastTotal(20000))
	sees(t, out, code, 0, "  running: slow (limit 20s)")
	streamedLive(t, out)
}

// timeoutNamesLastLine asserts a stopped check's row shows its last stderr
// line, named once at half its limit off a terminal.
func timeoutNamesLastLine(t *testing.T) {
	t.Helper()
	out, code := slowCheck(t, "exec /bin/sleep 30", fastTotal(1000))
	sees(t, out, code, 1, "Stopped at the 1s fast total", "Last output: "+slowFirst)
	if n := strings.Count(out, "running: slow (limit"); n != 1 {
		t.Errorf("off a terminal, a check past half its limit must be named once, but was named %d times:\n%s", n, out)
	}
}

// hugeTreeHints asserts that in 20,000 tracked files a scoped commit passes.
// An unscoped run stops at the fast total, its rows naming the count.
func hugeTreeHints(t *testing.T) {
	t.Helper()
	repo := generatedTree(t, staleTest(), 20000)
	out, code, _ := fixingCommit(t, repo)
	sees(t, out, code, 0)
	// Only the two judged checks run, so both start on a runner with few cores.
	out, code = fitness(t, repo, []string{fastTotal(100)}, "--policy=external", "--checks=cspell,duplicate-code")
	hint := fmt.Sprintf("Hint: the repo tracks %d files; scope the run with fitness --dir=<folder>, or commit to check only the staged files", trackedCount(t, repo))
	sees(t, out, code, 1, "Stopped at the 0.1s fast total", hint)
	for _, check := range []string{"cspell", "duplicate-code"} {
		if !strings.Contains(row(out, check), "failed") {
			t.Errorf("%s reads all 20,000 files, so a 0.1s fast total must stop it, got row %q", check, row(out, check))
		}
	}
}

// trackedCount is how many files git tracks in repo.
func trackedCount(t *testing.T, repo string) int {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "ls-files", "-z").Output()
	mustDo(t, err)
	return strings.Count(string(out), "\x00")
}

// progressOn asks for the heartbeat, as a terminal would.
const progressOn = "FITNESS_PROGRESS=1"

// slowFirst and slowLast are the stderr lines slowCheck's check writes
// before and after it waits.
const slowFirst, slowLast = "compiling 1 of 3", "compiling 3 of 3"

// slowCheck runs a local check, slow, beside a quick one and returns the output
// and exit code. Slow writes slowFirst to stderr, runs wait, then writes
// slowLast and passes.
func slowCheck(t *testing.T, wait string, env ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{
		".fitnessrc.json": `{"localChecks": "checks"}` + "\n",
		"checks/fitness-check-slow": "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"slow\"}'; exit 0; }\n" +
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

// childStopped asserts nothing the hung prettier started still runs. Under
// load its 0.3s budget can end before the check starts prettier at all, and
// then nothing ran to stop.
func childStopped(t *testing.T, repo string) {
	t.Helper()
	stopped(t, filepath.Join(repo, "prettier.pid"), 3*time.Second)
}

// processStopped asserts a fixture tool started, and that neither it nor any
// process it started still runs.
func processStopped(t *testing.T, pidFile string) {
	t.Helper()
	if !stopped(t, pidFile, 3*time.Second) {
		t.Fatalf("the tool that writes %s never started, so nothing shows fitness stops it", filepath.Base(pidFile))
	}
}

// stopped waits up to within until no process running finds for pidFile
// remains, failing if one outlasts it. It reports whether the tool wrote
// pidFile.
func stopped(t *testing.T, pidFile string, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	left := running(t, pidFile)
	for len(left) > 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		left = running(t, pidFile)
	}
	for _, pid := range left {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	if len(left) > 0 {
		t.Fatalf("processes %v from %s are still running %s after fitness stopped them", left, filepath.Base(pidFile), within)
	}
	return pidWritten(pidFile)
}

// pidWritten reports whether the tool wrote pidFile, or only its .tmp file when
// it was killed before moving it into place.
func pidWritten(pidFile string) bool {
	_, err := os.Stat(pidFile)
	_, errTmp := os.Stat(pidFile + ".tmp")
	return err == nil || errTmp == nil
}

// running lists the live processes whose ID pidFile holds, or whose command
// line names its folder. A fixture tool's does until it writes pidFile.
func running(t *testing.T, pidFile string) []int {
	t.Helper()
	pids := naming(t, filepath.Dir(pidFile))
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return pids
	}
	if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && sleeping(pid) && !slices.Contains(pids, pid) {
		pids = append(pids, pid)
	}
	return pids
}

// sleeping reports whether pid still runs a fixture tool's sleep. A process
// ID the system has reused is never counted.
func sleeping(pid int) bool {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	return err == nil && strings.Contains(string(out), "/bin/sleep")
}

// naming lists the processes whose command line names dir.
func naming(t *testing.T, dir string) []int {
	t.Helper()
	out, err := exec.Command("ps", "-A", "-ww", "-o", "pid=", "-o", "args=").Output()
	mustDo(t, err)
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		id, args, _ := strings.Cut(strings.TrimSpace(line), " ")
		if pid, err := strconv.Atoi(id); err == nil && strings.Contains(args, dir) {
			pids = append(pids, pid)
		}
	}
	return pids
}

// readme is happyRepo's README, for Givens that add to it.
const readme = "---\ntitle: App\n---\n\n# App\n\nShort and clean.\n"

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

// stalledGitStops asserts a stalled git file listing ends at its ten-second
// deadline as a setup error, and the git is stopped.
func stalledGitStops(t *testing.T) {
	t.Helper()
	for call, named := range map[string]string{
		"ls-files": "git ls-files -z --stage did not finish within 10s",
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
