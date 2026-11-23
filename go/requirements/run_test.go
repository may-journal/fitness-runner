// cspell:ignore zorbleflux

package requirements

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

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
	linkTools(bundle, installed, []string{"fitness"})
	placeRunner(t, filepath.Join(installed, "fitness"), filepath.Join(bundle, "fitness"))
	write(t, bundle, map[string]string{bin: "#!/bin/sh\n[ \"$1\" = --describe ] && exit 1\nexec \"" + installedCheck(t, check) + "\" \"$@\"\n"})
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), check)
		// The wrapper beside fitness must win over the installed folder.
		cmd.Dir, cmd.Env = repo, append(userEnv(), "FITNESS_CHECKS_DIR=")
		return cmd
	})
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

// timedPrefix names the slow checks Test0003_4 installs. Run under such a
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

// inActions runs the one-liner in repo as a GitHub Actions job does, returning
// its output, exit code and job summary.
func inActions(t *testing.T, repo string) (string, int, string) {
	t.Helper()
	summary := filepath.Join(t.TempDir(), "summary.md")
	out, code := fitness(t, repo, []string{"GITHUB_ACTIONS=true", "GITHUB_WORKSPACE=" + repo, "GITHUB_STEP_SUMMARY=" + summary})
	written, _ := os.ReadFile(summary)
	return out, code, string(written)
}

// misspelled is happyRepo's README with n unknown words, one per line.
func misspelled(n int) map[string]string {
	body := readme
	for i := 0; i < n; i++ {
		body += "\nThe zorbleflux sits here.\n"
	}
	return map[string]string{"README.md": body}
}

// runAlone runs the runner at alone in repo with args, as a user types it.
func runAlone(t *testing.T, alone, repo string, args ...string) (string, int) {
	t.Helper()
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(alone, args...)
		cmd.Dir, cmd.Env = repo, userEnv()
		return cmd
	})
}

// perfCheck is one check's entry in the `--json` report, as a reader of the
// file sees it.
type perfCheck struct {
	Name         string `json:"name"`
	Ok           bool   `json:"ok"`
	Ms           int64  `json:"ms"`
	Stage        string `json:"stage"`
	TimedOut     bool   `json:"timedOut"`
	FilesChecked int    `json:"filesChecked"`
	BinaryBytes  int64  `json:"binaryBytes"`
	ErrorLines   int    `json:"errorLines"`
}

// perfReport is the `--json` report file.
type perfReport struct {
	ElapsedMs   int64       `json:"elapsedMs"`
	RunnerBytes int64       `json:"runnerBytes"`
	Checks      []perfCheck `json:"checks"`
}

// withReport runs the full suite with `--json` in repo and returns what it
// printed and the report it wrote.
func withReport(t *testing.T, repo string, env ...string) (string, perfReport) {
	t.Helper()
	return selectedReport(t, repo, env, "--all")
}

// selectedReport is withReport for the checks selection names.
func selectedReport(t *testing.T, repo string, env []string, selection string) (string, perfReport) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "perf", "fitness.json")
	out, _ := fitness(t, repo, env, "--policy=external", selection, "--json="+path)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fitness --json wrote no report: %v\n%s", err, out)
	}
	var report perfReport
	mustDo(t, json.Unmarshal(raw, &report))
	return out, report
}

// reported returns the report's entry for check.
func reported(t *testing.T, report perfReport, check string) perfCheck {
	t.Helper()
	for _, c := range report.Checks {
		if c.Name == check {
			return c
		}
	}
	t.Fatalf("the report has no entry for %s", check)
	return perfCheck{}
}

// verdicts asserts the misspelling failed cspell while prose-budget passed.
func verdicts(t *testing.T, report perfReport) {
	t.Helper()
	if cspell := reported(t, report, "cspell"); verdictOf(cspell) != (verdict{ok: false, judged: true, errors: 1}) {
		t.Errorf("cspell must fail with one error having judged files, got %+v", cspell)
	}
	if prose := reported(t, report, "prose-budget"); verdictOf(prose) != (verdict{ok: true, judged: prose.FilesChecked > 0, errors: 0}) {
		t.Errorf("prose-budget must pass with no errors, got %+v", prose)
	}
}

// verdict is what a reader takes from one report entry: the result, whether
// files were judged, and how many errors it found.
type verdict struct {
	ok, judged bool
	errors     int
}

func verdictOf(c perfCheck) verdict {
	return verdict{ok: c.Ok, judged: c.FilesChecked > 0, errors: c.ErrorLines}
}

// fullEntry asserts a report entry matches a table row and carries its
// stage, binary size, and time.
func fullEntry(t *testing.T, out string, c perfCheck) {
	t.Helper()
	if row(out, c.Name) == "" {
		t.Errorf("the report lists %s, which the table never showed", c.Name)
	}
	if c.Stage == "" || c.BinaryBytes <= 0 || c.Ms < 0 {
		t.Errorf("%s must carry its stage, binary size, and time, got %+v", c.Name, c)
	}
}

// timingRow matches one line of the job summary's timing table.
var timingRow = regexp.MustCompile(`^\| (\S+) \| (fast|tests) \| (\d+)ms \|$`)

// timings returns the times in the summary's timing table, in order.
func timings(summary string) []int64 {
	var times []int64
	for _, line := range strings.Split(summary, "\n") {
		if m := timingRow.FindStringSubmatch(line); m != nil {
			ms, _ := strconv.ParseInt(m[3], 10, 64)
			times = append(times, ms)
		}
	}
	return times
}

// diagramBox matches one check's box in the page: its result class, then its
// tooltip naming the check, its value, and its result.
var diagramBox = regexp.MustCompile(`<li class="([a-z-]*)"[^>]*><div class="cell" tabindex="0" title="([^:"]+): ([^,"]+), ([a-z ]+)">`)

// outsideRequest matches anything that would make the page run code or load
// another file.
var outsideRequest = regexp.MustCompile(`<script|http|src=`)

// diagramViews are the page's three ways to size the boxes.
var diagramViews = []string{"time", "size", "errors"}

// drawn writes the report to a file, runs `fitness boxes` on it the way a
// user does, and returns the page.
func drawn(t *testing.T, repo string, report perfReport) string {
	t.Helper()
	dir := t.TempDir()
	in, out := filepath.Join(dir, "fitness.json"), filepath.Join(dir, "page", "fitness.html")
	raw, err := json.Marshal(report)
	mustDo(t, err)
	mustDo(t, os.WriteFile(in, raw, 0o644))
	said, code := fitness(t, repo, nil, "boxes", in, out)
	sees(t, said, code, 0)
	page, err := os.ReadFile(out)
	mustDo(t, err)
	return string(page)
}

// oneBoxEach asserts a view draws every reported check once, with its result
// in words and a matching class.
func oneBoxEach(t *testing.T, name, view string, report perfReport) {
	t.Helper()
	seen := map[string]int{}
	for _, m := range diagramBox.FindAllStringSubmatch(view, -1) {
		seen[m[2]]++
		saysResult(t, name, m[1], m[4], reported(t, report, m[2]))
	}
	for _, c := range report.Checks {
		if seen[c.Name] != 1 {
			t.Errorf("%s view draws %s %d times, want once", name, c.Name, seen[c.Name])
		}
	}
}

// saysResult asserts a box names its check's result in words, and carries a
// failure class exactly when the check did not pass.
func saysResult(t *testing.T, view, class, words string, c perfCheck) {
	t.Helper()
	if got, want := [2]any{words, class == ""}, [2]any{resultWords(c), c.Ok}; got != want {
		t.Errorf("%s view: %s box says %q with class %q, want %q", view, c.Name, words, class, want[0])
	}
}

// resultWords is how the page names a check's result.
func resultWords(c perfCheck) string {
	switch {
	case c.TimedOut:
		return "timed out"
	case c.Ok:
		return "passed"
	}
	return "failed"
}

// releaseRecord is what `fitness-release record` prints.
type releaseRecord struct {
	Version   string `json:"version"`
	Platform  string `json:"platform"`
	InstallMs int64  `json:"installMs"`
	Assets    []struct {
		Name  string `json:"name"`
		Bytes int64  `json:"bytes"`
	} `json:"assets"`
}

// recorded runs `fitness-release record` in repo and decodes what it printed.
func recorded(t *testing.T, repo string) releaseRecord {
	t.Helper()
	out, code := releaseToolRun(t, repo, nil, "record")
	var record releaseRecord
	start, end := strings.Index(out, "{"), strings.LastIndex(out, "}")
	if code != 0 || start < 0 || json.Unmarshal([]byte(out[start:end+1]), &record) != nil {
		t.Fatalf("record must exit 0 printing JSON, got exit %d:\n%s", code, out)
	}
	return record
}

// describesHost asserts the record names this platform, the version, and an
// install time.
func describesHost(t *testing.T, record releaseRecord) {
	t.Helper()
	if record.Platform != runtime.GOOS+"-"+runtime.GOARCH || record.InstallMs <= 0 || record.Version == "" {
		t.Errorf("record must name this platform, the version, and an install time, got %+v", record)
	}
}

// realSize asserts the file at path is bytes long.
func realSize(t *testing.T, path string, bytes int64) {
	t.Helper()
	if info, err := os.Stat(path); err != nil || info.Size() != bytes {
		t.Errorf("%s must be recorded at its real size, got %d bytes", filepath.Base(path), bytes)
	}
}

// dirRepo is happyRepo with a badly named doc in each of two folders.
func dirRepo(t *testing.T) string {
	t.Helper()
	return example(t, "happyRepo", kebabDocsRepo(map[string]string{
		"docs/releaseNotes.md": "# Notes\n",
		"guides/userGuide.md":  "# Guide\n",
	}))
}

// finding is one structured finding, as a reader of `--format=json` sees it.
type finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Fix     string `json:"fix"`
	Explain string `json:"explain"`
}

// jsonRun is what `fitness --format=json` prints on stdout.
type jsonRun struct {
	Ok     bool `json:"ok"`
	Checks []struct {
		Name     string    `json:"name"`
		Status   string    `json:"status"`
		Errors   []string  `json:"errors"`
		Findings []finding `json:"findings"`
	} `json:"checks"`
}

// asJSON runs the suite with `--format=json` in repo as an agent does,
// keeping stdout apart from the stderr table, and decodes stdout.
func asJSON(t *testing.T, repo string) jsonRun {
	t.Helper()
	var stdout, stderr bytes.Buffer
	started(t, func() *exec.Cmd {
		stdout.Reset()
		stderr.Reset()
		cmd := exec.Command(installer(t), "--", "--policy=external", "--all", "--format=json")
		cmd.Dir, cmd.Env = repo, userEnv()
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		return cmd
	}, (*exec.Cmd).Run)
	var run jsonRun
	if err := json.Unmarshal(stdout.Bytes(), &run); err != nil {
		t.Fatalf("stdout must be one JSON object: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return run
}

// findingsOf returns check's findings in run, failing when it did not run.
func findingsOf(t *testing.T, run jsonRun, check string) []finding {
	t.Helper()
	for _, c := range run.Checks {
		if c.Name == check {
			return c.Findings
		}
	}
	t.Fatalf("the JSON lists no %s check", check)
	return nil
}

// oneFinding asserts check reported exactly one finding at file and line,
// with rule, a fix, and the command that explains check.
func oneFinding(t *testing.T, run jsonRun, check, file string, line int, rule string) {
	t.Helper()
	got := findingsOf(t, run, check)
	if len(got) != 1 {
		t.Fatalf("%s must report one finding, got %+v", check, got)
	}
	f := got[0]
	fixed := f.Fix != ""
	f.Message, f.Fix = "", ""
	if want := (finding{File: file, Line: line, Rule: rule, Explain: "fitness explain " + check}); f != want || !fixed {
		t.Errorf("%s finding must name %s:%d, rule %s, a fix, and its explain command, got %+v", check, file, line, rule, got[0])
	}
}

// listedCheck is one check as `fitness list --json` gives it.
type listedCheck struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Scope   string `json:"scope"`
	Stage   string `json:"stage"`
	Enabled bool   `json:"enabled"`
}

// listJSON runs `fitness list --json` in repo and decodes what it printed
// after the installer's own lines.
func listJSON(t *testing.T, repo string) []listedCheck {
	t.Helper()
	out, code := fitness(t, repo, nil, "list", "--json")
	if code != 0 {
		t.Fatalf("fitness list --json exited %d:\n%s", code, out)
	}
	var checks []listedCheck
	start := strings.Index(out, "\n[")
	if start < 0 {
		t.Fatalf("fitness list --json printed no check list:\n%s", out)
	}
	if err := json.Unmarshal([]byte(out[start+1:]), &checks); err != nil {
		t.Fatalf("fitness list --json printed no check list: %v\n%s", err, out)
	}
	return checks
}

// listRow returns the `fitness list` line for check, or "".
func listRow(out, check string) string {
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == check {
			return line
		}
	}
	return ""
}

// described reports whether a `fitness list` row shows a scope, a stage, and
// a summary.
func described(row string) bool {
	fields := strings.Fields(row)
	return len(fields) >= 5 && fields[1] != "-" && (fields[2] == "fast" || fields[2] == "tests") && fields[4] != "-"
}

// runnerGoneAtBuild runs `fitness list` from a bundle in a repo with a Go check.
// Its build, before any handshake, runs a go that removes the bundle's runner.
func runnerGoneAtBuild(t *testing.T) (string, int) {
	t.Helper()
	installed := installOnly(t)
	bundle, tools := t.TempDir(), t.TempDir()
	runner := filepath.Join(bundle, "fitness")
	placeRunner(t, filepath.Join(installed, "fitness"), runner)
	real, err := exec.LookPath("go")
	mustDo(t, err)
	write(t, tools, map[string]string{"go": "#!/bin/sh\nrm -f \"" + runner + "\"\nexec \"" + real + "\" \"$@\"\n"})
	repo := example(t, "happyRepo", goCheck(t, "go-rule", t.Name()+time.Now().String()))
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(runner, "list")
		cmd.Dir, cmd.Env = repo, append(userEnv(), "PATH="+tools+string(os.PathListSeparator)+os.Getenv("PATH"))
		return cmd
	})
}

// explainedCheck is one check as `fitness explain --json` gives it.
type explainedCheck struct{ Name, Summary, Scope, Body string }

// explainJSON runs `fitness explain <check> --json` in repo and decodes what
// it printed after the installer's own lines.
func explainJSON(t *testing.T, repo, check string) explainedCheck {
	t.Helper()
	out, code := fitness(t, repo, nil, "explain", check, "--json")
	if code != 0 {
		t.Fatalf("fitness explain --json exited %d:\n%s", code, out)
	}
	var got explainedCheck
	start := strings.Index(out, "\n{")
	if start < 0 {
		t.Fatalf("fitness explain --json printed no object:\n%s", out)
	}
	if err := json.Unmarshal([]byte(out[start+1:]), &got); err != nil {
		t.Fatalf("fitness explain --json printed no object: %v\n%s", err, out)
	}
	return got
}

// opening is the first four words of a summary, which a check's Markdown
// documentation opens with, once its escapes are dropped.
func opening(summary string) string {
	words := strings.Fields(summary)
	return strings.Join(words[:min(4, len(words))], " ")
}

// listedSummary is the summary `fitness list --json` gives check in repo;
// a check it lists with none fails the test.
func listedSummary(t *testing.T, repo, check string) string {
	t.Helper()
	for _, c := range listJSON(t, repo) {
		if c.Name == check && c.Summary != "" {
			return c.Summary
		}
	}
	t.Fatalf("fitness list --json gives %s no summary", check)
	return ""
}

// opensWith reports whether a Markdown body, once its escapes are dropped,
// opens with summary's first words and says more than summary alone.
func opensWith(body, summary string) bool {
	plain := strings.ReplaceAll(body, `\`, "")
	return strings.HasPrefix(plain, opening(summary)) && len(plain) > len(summary)+1
}

// printsTopicFile fails unless `fitness docs <name>` prints go/docs/<name>.md
// titled with the name, and list, what `fitness docs` printed, shows its
// opening.
func printsTopicFile(t *testing.T, repo, list, name string) {
	t.Helper()
	file := repoFile(t, "go/docs/"+name+".md")
	page, code := fitness(t, repo, nil, "docs", name)
	if code != 0 || !strings.HasSuffix("\n"+page, "\n"+file) || !strings.HasPrefix(file, "# "+name+"\n") {
		t.Errorf("fitness docs %s (exit %d) does not print go/docs/%s.md, title first:\n%s", name, code, name, page)
	}
	listsOpening(t, list, name, file)
}

// listsOpening fails unless list shows name with the first paragraph after
// file's title, one sentence on one line.
func listsOpening(t *testing.T, list, name, file string) {
	t.Helper()
	_, rest, _ := strings.Cut(file, "\n\n")
	first, _, _ := strings.Cut(rest, "\n\n")
	if sentence := strings.Join(strings.Fields(first), " "); !strings.Contains(list, sentence+"\n") || strings.Contains(sentence, ". ") {
		t.Errorf("fitness docs does not list %s with the one sentence its file opens with, %q:\n%s", name, sentence, list)
	}
}

// topicNames reads the topic names `fitness docs` lists: the first word of
// each line after the installer's own lines.
func topicNames(out string) []string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) > 1 && !strings.HasPrefix(line, "fitness") {
			names = append(names, fields[0])
		}
	}
	return names
}

// subcommands are the commands fitness runs besides checks.
var subcommands = strings.Fields(`llms list explain docs check init hook boxes
strip-front-matter branch-check pr-check issue-check plan-check close-check
dupe-check comment-check`)

// repoFile reads a file of the fitness repository these tests live in.
func repoFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(path)))
	mustDo(t, err)
	return string(data)
}

// commentLine matches a line `fitness comments` prints for one block.
var commentLine = regexp.MustCompile(`^\S+:\d+: \S+ (doc|inline): `)

// listComments writes files over happyRepo and runs `fitness comments`
// on the source files among them.
func listComments(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	var paths []string
	for _, p := range slices.Sorted(maps.Keys(files)) {
		if p != ".gitattributes" {
			paths = append(paths, p)
		}
	}
	return fitness(t, repo, nil, append([]string{"comments"}, paths...)...)
}

// listsExactly asserts the run exits 0 and its comment lines are want, in
// order.
func listsExactly(t *testing.T, out string, code int, want ...string) {
	t.Helper()
	var got []string
	for _, line := range strings.Split(out, "\n") {
		if commentLine.MatchString(line) {
			got = append(got, line)
		}
	}
	if code != 0 || !slices.Equal(got, want) {
		t.Errorf("exit %d, listed:\n%s\nwant:\n%s\noutput:\n%s", code, strings.Join(got, "\n"), strings.Join(want, "\n"), out)
	}
}

// backtick spells the quote that Go raw strings cannot hold.
const backtick = "`"

// viewsPartner is a file check that passes on happyRepo. A run of it and
// one more check reads both views the runner built for them.
const viewsPartner = "no-plans-dir"

// withPartner runs check and viewsPartner together on repo.
func withPartner(t *testing.T, repo, check string) (string, int) {
	t.Helper()
	return fitness(t, repo, nil, "--policy=external", "--checks="+check+","+viewsPartner)
}

// stopsUnderLoad asserts a hung prettier, after prelude, ends within 1s past
// a 2s fast total, and no tool runs once fitness exits.
func stopsUnderLoad(t *testing.T, prelude string) {
	t.Helper()
	repo := example(t, "happyRepo", hungTool(prelude))
	path := filepath.Join(t.TempDir(), "fitness.json")
	out, code := fitness(t, repo, []string{fastTotal(2000)}, "--policy=external", "--checks=prettier", "--json="+path)
	sees(t, out, code, 1, "Stopped at the 2s fast total")
	raw, err := os.ReadFile(path)
	mustDo(t, err)
	var report perfReport
	mustDo(t, json.Unmarshal(raw, &report))
	if past := time.Duration(reported(t, report, "prettier").Ms)*time.Millisecond - 2*time.Second; past > time.Second {
		t.Errorf("with every CPU busy, prettier ended %s past the 2s fast total, want within 1s", past)
	}
	if !stopped(t, filepath.Join(repo, "prettier.pid"), 200*time.Millisecond) {
		t.Errorf("prettier never started within the 2s fast total, so nothing shows its child stops")
	}
}

// busyCPUs keeps every CPU busy with one spinning shell per CPU until the
// test ends.
func busyCPUs(t *testing.T) {
	t.Helper()
	for range runtime.NumCPU() {
		cmd := exec.Command("/bin/sh", "-c", "while :; do :; done")
		mustDo(t, cmd.Start())
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		})
	}
}

// perfRow matches one row of `fitness perf`: check, stage, time and result.
var perfRow = regexp.MustCompile(`^(\S+)\s+(fast|tests)\s+([\d.]+)(m?s)\s+(passed|failed|timed out)$`)

// moduleSleep is how long each module's one test sleeps in Test0003_4.
const moduleSleep = 6 * time.Second

// fastNames are the fast checks Test0003_4 runs beside a slow test runner.
var fastNames = []string{"quick1", "quick2", "quick3", "quick4", "quick5", "quick6"}

// scriptApplies is a check program whose --describe prints describe and whose
// run fails saying it ran, so the summary shows which checks ran.
func scriptApplies(name, describe string) string {
	return "#!/bin/sh\n[ \"$1\" = --describe ] && { printf '%s\\n' '" + describe + "'; exit 0; }\n" +
		"echo '{\"ok\":false,\"errors\":[\"" + name + " ran\"],\"filesChecked\":1}'\nexit 1\n"
}

// runApplies runs the fitness in bundle in repo with stdin and args.
func runApplies(t *testing.T, bundle, repo, stdin string, args ...string) (string, int) {
	t.Helper()
	return execute(t, func() *exec.Cmd {
		cmd := exec.Command(filepath.Join(bundle, "fitness"), args...)
		cmd.Dir, cmd.Env = repo, append(userEnv(), "FITNESS_CHECKS_DIR=")
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		return cmd
	})
}

// absent fails when out names a check as having run.
func absent(t *testing.T, out string, ran ...string) {
	t.Helper()
	for _, name := range ran {
		if strings.Contains(out, name+" ran") {
			t.Errorf("%s ran, but the run should leave it out:\n%s", name, out)
		}
	}
}

// bodyOnlyPaths republishes r's index with the Markdown and Go checks and
// open-questions, a check local-branch needs, whose files find only bodies.
func bodyOnlyPaths(t *testing.T, r *releasesOnDemand) {
	t.Helper()
	r.archives["open-questions"] = archiveOnDemand(t, "open-questions")
	r.publish(t, map[string]string{"md-files": "1.0.0", "go-files": "1.0.0", "open-questions": "1.0.0"}, time.Now())
	path := filepath.Join(r.cache, "check-index.json")
	data, err := os.ReadFile(path)
	mustDo(t, err)
	var record struct {
		Value struct {
			Format int                                   `json:"format"`
			Checks map[string]map[string]json.RawMessage `json:"checks"`
		} `json:"value"`
		Checked string `json:"checked"`
	}
	mustDo(t, json.Unmarshal(data, &record))
	var versions map[string]map[string]any
	mustDo(t, json.Unmarshal(record.Value.Checks["open-questions"]["versions"], &versions))
	versions["1.0.0"]["files"] = `^\.fitness/`
	patched, err := json.Marshal(versions)
	mustDo(t, err)
	record.Value.Checks["open-questions"]["versions"] = patched
	data, err = json.Marshal(record)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o600))
}

// checksPaths are installed check programs, by name, whose --describe
// declares the old scope and blob fields, with or without their own files.
var checksPaths = map[string]string{
	"history-rule": `{"name":"history-rule","scope":"git"}`,
	"body-rule":    `{"name":"body-rule","scope":"blob","blob":true}`,
	"md-body":      `{"name":"md-body","scope":"blob","blob":true,"files":"\\.md$"}`,
	"go-text":      `{"name":"go-text","blob":true,"files":"\\.go$"}`,
	"go-files":     `{"name":"go-files","files":"\\.go$"}`,
	"git-text":     `{"name":"git-text","scope":"git","blob":true}`,
	"no-notes":     `{"name":"no-notes","exclude":"^notes/"}`,
}

// bundlePaths is a fitness bundle holding every check in checksPaths.
func bundlePaths(t *testing.T) string {
	t.Helper()
	bundle := bundleDiscovery(t)
	programs := map[string]string{}
	for name, describe := range checksPaths {
		programs["fitness-check-"+name] = scriptApplies(name, describe)
	}
	write(t, bundle, programs)
	return bundle
}
