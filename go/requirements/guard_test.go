// cspell:ignore aproved approvd apprvoed dont
// cspell:ignore mispeled

package requirements

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// installerEnv puts the installer where a user's setup does, without fitness on
// the PATH, and returns the environment the hooks need.
func installerEnv(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	mustDo(t, os.Symlink(installer(t), filepath.Join(dir, "fitness-install")))
	lookups := rlReleases(t, http.StatusOK, rlCandidate(t))
	return slices.Concat([]string{"FITNESS_INSTALL_DIR=" + dir}, lookups.env, seededChecks(t, fiPrivateCache(t)))
}

// hooked copies happyRepo with files over it, runs `fitness init` with the
// installer placed by installerEnv, and returns the repo and that environment.
func hooked(t *testing.T, files map[string]string) (string, []string) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	env := installerEnv(t)
	got, code := fitness(t, repo, env, "init")
	sees(t, got, code, 0, "fitness init: installed hooks")
	return repo, env
}

// user runs git in repo as a person at the terminal does, with hooks, returning
// its output and exit code.
func user(t *testing.T, repo string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	cmd.Dir, cmd.Env = repo, append(userEnv(), env...)
	out, _ := cmd.CombinedOutput()
	return ansi.ReplaceAllString(string(out), ""), cmd.ProcessState.ExitCode()
}

// remote gives repo an empty bare origin to push to, so no test needs GitHub.
func remote(t *testing.T, repo string) {
	t.Helper()
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "add", "origin", bare)
}

// localHookStops asserts a pre-commit.local ignoring the stop signal, with git
// awaiting a password, stops at a 2s test total. The commit is refused.
func localHookStops(t *testing.T) {
	t.Helper()
	repo, env := hooked(t, nil)
	env = append(env, testsTotal(2000))
	git(t, repo, "remote", "add", "origin", promptedRemote)
	path, pidFile := promptingTools(t, "fetch")
	// Git puts its own folder first on a hook's PATH, so the hook names the waiting git.
	tools := strings.SplitN(strings.TrimPrefix(path, "PATH="), string(os.PathListSeparator), 2)[0]
	write(t, repo, map[string]string{".githooks/pre-commit.local": "#!/bin/sh\ntrap '' TERM\n\"" + tools + "/git\" fetch origin\n"})
	out, code := user(t, repo, append(env, path, "GIT_TERMINAL_PROMPT=0"), "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 1, "fitness hook: .githooks/pre-commit.local timed out after 2s")
	processStopped(t, pidFile)
}

// Comment authors on the saved Plans: the repo's owner, a passer-by, an org
// member, and a reviewer listed in planApprovers.
var (
	owner    = func(body string) comment { return comment{Body: body, Author: "example-owner", Association: "OWNER"} }
	passerby = func(body string) comment { return comment{Body: body, Author: "passerby", Association: "NONE"} }
	member   = func(body string) comment { return comment{Body: body, Author: "org-member", Association: "MEMBER"} }
	reviewer = func(body string) comment { return comment{Body: body, Author: "reviewer-bot", Association: "NONE"} }
)

// pushPlans are saved GitHub answers about issues a PR may close. #218 is
// owner-approved, #196 a bug, #151 passer-by, #307 reviewer, #308 member.
// #309 to #320 say approve in other ways, #321 to #328 and #330 do not, and #329 is a passer-by.
func pushPlans(t *testing.T) []response {
	t.Helper()
	plan, labels := planBody("- [ ] Ship it."), []string{"Plan"}
	return []response{
		planViewBy(t, 218, plan, labels, passerby("Looks good."), owner("I approve this Plan.")),
		planViewBy(t, 151, plan, []string{"Plan", "enhancement"}, owner("This is not approved; it needs work."), passerby("I approve.")),
		planView(t, 196, "The parser drops the last line.", []string{"bug"}, "I approve."),
		planViewBy(t, 307, plan, labels, reviewer("I approve.")),
		planViewBy(t, 308, plan, labels, member("i approve, go ahead")),
		planViewBy(t, 309, plan, labels, owner("I approved.")),
		planViewBy(t, 310, plan, labels, owner("Approved")),
		planViewBy(t, 311, plan, labels, owner("LGTM, approved")),
		planViewBy(t, 312, plan, labels, owner("Approving this, go ahead.")),
		planViewBy(t, 313, plan, labels, owner("Plan approval: yes")),
		planViewBy(t, 314, plan, labels, owner("aproved, thanks")),
		planViewBy(t, 315, plan, labels, owner("approvd")),
		planViewBy(t, 316, plan, labels, owner("apprvoed")),
		planViewBy(t, 317, plan, labels, owner("Approved!")),
		planViewBy(t, 318, plan, labels, owner("I APPROVE")),
		planViewBy(t, 319, plan, labels, owner("approves")),
		planViewBy(t, 320, plan, labels, owner("Ok. I approve")),
		planViewBy(t, 321, plan, labels, owner("Disapprove")),
		planViewBy(t, 322, plan, labels, owner("I dont approve")),
		planViewBy(t, 323, plan, labels, owner("This is not approved")),
		planViewBy(t, 324, plan, labels, owner("never approved")),
		planViewBy(t, 325, plan, labels, owner("unapproved")),
		planViewBy(t, 326, plan, labels, owner("The approach is sound.")),
		planViewBy(t, 327, plan, labels, owner("Roughly approximately right.")),
		planViewBy(t, 328, plan, labels, owner("No approval.")),
		planViewBy(t, 330, plan, labels, owner("I can't approve this, not yet approved")),
		planViewBy(t, 329, plan, labels, passerby("I approved.")),
	}
}

// gateOnly is the config of a repo whose push asks GitHub only what the gate
// asks, since local-branch is off.
const gateOnly = `{"policy": "external", "disabledChecks": ["local-branch"]}` + "\n"

// listed is gateOnly whose planApprovers names only the reviewer, in another
// case than GitHub reports the login.
const listed = `{"policy": "external", "disabledChecks": ["local-branch"], "planApprovers": ["Reviewer-Bot"]}` + "\n"

// pushEnv adds the stand-ins, first on PATH, to the environment that already holds the installer.
func pushEnv(hookEnv []string, rp replays) []string {
	return slices.Concat(hookEnv, rp.env)
}

// githubRepo is a hooked repo with config rc whose origin is repoPR on
// GitHub, pushed to a bare folder instead.
func githubRepo(t *testing.T, rc string) (string, []string) {
	t.Helper()
	repo, hookEnv := hooked(t, map[string]string{".fitnessrc.json": rc})
	git(t, repo, "remote", "add", "origin", "https://github.com/"+repoPR+".git")
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "set-url", "--push", "origin", bare)
	return repo, hookEnv
}

// pushRepo is a githubRepo on topic, after pushing main. Its gh stand-in
// finds topic's open PR with body, or none for an empty body.
func pushRepo(t *testing.T, rc, body string) (string, []string, replays) {
	t.Helper()
	repo, hookEnv := githubRepo(t, rc)
	saved := []response{{Match: []string{"repo view --json nameWithOwner"}, Stdout: repoPR + "\n"}}
	if body != "" {
		saved = append(saved, pushedAnswer(t, repo, "topic", "feat(app): add a feature", body, 1))
	}
	saved = append(saved, response{Match: []string{"pulls?state=open"}})
	rp := standIn(t, map[string][]response{"gh": slices.Concat(saved, pushPlans(t), orgPlans(t))})
	env := pushEnv(hookEnv, rp)
	if out, code := user(t, repo, env, "push", "-q", "origin", "HEAD:main"); code != 0 {
		t.Fatalf("pushing main: %s", out)
	}
	user(t, repo, env, "switch", "-q", "-c", "topic")
	return repo, env, rp
}

// pushFeature commits a feature with message on topic, whose open PR has
// body, and pushes it.
func pushFeature(t *testing.T, rc, body, message string) (string, int, replays) {
	t.Helper()
	repo, env, rp := pushRepo(t, rc, body)
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", message)
	out, code := user(t, repo, env, "push", "origin", "topic")
	return out, code, rp
}

// pushClosing pushes a feature commit on topic, whose open PR has body.
func pushClosing(t *testing.T, body string) (string, int, replays) {
	t.Helper()
	return pushFeature(t, gateOnly, body, "feat(app): add a feature")
}

// pushAsked fails t unless the hook asked GitHub about issue n's field.
func pushAsked(t *testing.T, rp replays, n, field string) {
	t.Helper()
	if !rp.called("gh", "issue view "+n+" ", field) {
		t.Errorf("pre-push never read issue %s's %s; gh calls: %q", n, field, rp.calls("gh"))
	}
}

// planReadOnce fails t unless the push asked GitHub about issue n exactly
// once, through the hook's read or an issue read.
func planReadOnce(t *testing.T, rp replays, n string) {
	t.Helper()
	got := 0
	for _, call := range rp.calls("gh") {
		if strings.Contains(call, "issue view "+n+" ") || strings.HasSuffix(call, "/issues/"+n) {
			got++
		}
	}
	if got != 1 {
		t.Errorf("want one read of issue %s, got %d; gh calls: %q", n, got, rp.calls("gh"))
	}
}

// silentGitHub is the environment of the real gh, with a token, behind a
// silent proxy.
func silentGitHub(t *testing.T) []string {
	t.Helper()
	if _, err := exec.LookPath("gh"); err != nil {
		t.Fatalf("this test runs the real gh: %v", err)
	}
	return []string{"HTTPS_PROXY=" + silentListener(t), "NO_PROXY=", "no_proxy=", "GH_TOKEN=unused", "GH_HOST=github.com",
		"GH_REPO=" + repoPR, "GH_CONFIG_DIR=" + t.TempDir(), "GH_NO_UPDATE_NOTIFIER=1"}
}

// silentPushRefused pushes a feature branch behind a GitHub that never
// answers. Each read once waited 30 seconds; now the first to run out ends
// the push with one line.
func silentPushRefused(t *testing.T) {
	t.Helper()
	repo, hookEnv := githubRepo(t, gateOnly)
	user(t, repo, hookEnv, "switch", "-q", "-c", "topic")
	out, code := user(t, repo, hookEnv, "commit", "--allow-empty", "-m", "feat(app): add a feature")
	sees(t, out, code, 0, "feat(app): add a feature")
	start := time.Now()
	out, code = user(t, repo, append(hookEnv, silentGitHub(t)...), "push", "origin", "topic")
	sees(t, out, code, 1, "pre-push: github.com did not answer within 30s; check the network or proxy, then push again.")
	if took := time.Since(start); took > 40*time.Second {
		t.Errorf("push took %s, past the 30-second deadline", took)
	}
	if n := strings.Count(out, "pre-push:"); n != 1 {
		t.Errorf("want one pre-push line, got %d:\n%s", n, out)
	}
}

// orgPlanView is GitHub's saved answer about issue n of repo, read with -R.
func orgPlanView(t *testing.T, repo string, n int, labels []string, comments ...string) response {
	t.Helper()
	r := planView(t, n, planBody("- [ ] Ship it."), labels, comments...)
	r.Stdout = strings.Replace(r.Stdout, repoPR, repo, 1)
	r.Match = append(r.Match, "-R "+repo)
	return r
}

// orgPlans are GitHub's saved answers about other repos' Plans: .github#144
// approved in any owner case, #141 not, other-org/.github#7 by another owner.
func orgPlans(t *testing.T) []response {
	t.Helper()
	return []response{
		orgPlanView(t, "may-journal/.github", 144, []string{"Plan"}, "I approve"),
		orgPlanView(t, "May-Journal/.github", 144, []string{"Plan"}, "I approve"),
		orgPlanView(t, "may-journal/.github", 141, []string{"Plan"}, "Could this wait a week?"),
		orgPlanView(t, "other-org/.github", 7, []string{"Plan"}, "I approve"),
	}
}

// linked is a hooked repo on branch topic, off a pushed main, whose one
// commit links issue #218.
func linked(t *testing.T) (string, []string) {
	t.Helper()
	repo, env := hooked(t, nil)
	remote(t, repo)
	user(t, repo, env, "push", "-q", "origin", "HEAD:main")
	user(t, repo, env, "switch", "-q", "-c", "topic")
	if out, code := user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "chore(app): start\n\nPlan #218"); code != 0 {
		t.Fatalf("first link: %s", out)
	}
	return repo, env
}

// offlineNote is the note the check carries when the probe found GitHub's
// API host silent behind the proxy. A missing gh skips the PR rule too, so
// the note proves the probe path.
const offlineNote = "api.github.com did not answer within 2s through proxy http://127.0.0.1:"

// silentCommit commits msg in repo behind a silent proxy with the real gh, and
// fails when the commit takes offlineWithin or more.
func silentCommit(t *testing.T, repo string, env []string, msg string) (string, int) {
	t.Helper()
	env = append(append(env, silentGitHub(t)...), "GITHUB_API_URL=https://api.github.com")
	start := time.Now()
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", msg)
	if took := time.Since(start); took >= offlineWithin {
		t.Errorf("commit took %s behind a silent proxy, want under %s:\n%s", took.Round(time.Second), offlineWithin, out)
	}
	return out, code
}

// branchRepo supplies a local branch and fetched base without network writes.
func branchRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", files)
	git(t, repo, "branch", "-M", "work")
	git(t, repo, "remote", "add", "origin", "https://github.com/"+repoPR+".git")
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	return repo
}

// branchAnswer is the saved result of the filtered open-PR request.
func branchAnswer(t *testing.T, repo, title, body string) response {
	t.Helper()
	base, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	mustDo(t, err)
	data, err := json.Marshal(map[string]any{
		"number": 1, "title": title, "body": body, "head": "work", "source": repoPR,
		"base": "main", "baseSHA": strings.TrimSpace(string(base)), "url": "https://github.com/" + repoPR + "/pull/1", "draft": true,
	})
	mustDo(t, err)
	return response{Match: []string{"pulls?state=open&head=" + url.QueryEscape("may-journal:work")}, Stdout: string(data) + "\n"}
}

func branchGitHub(pr response, issues ...response) []response {
	saved := []response{{Match: []string{"repo view --json nameWithOwner"}, Stdout: repoPR + "\n"}, pr}
	return append(saved, issues...)
}

// forkBranch has a stale fork main, an upstream Plan, and a branch-only Plan.
func forkBranch(t *testing.T, base string) (string, response) {
	t.Helper()
	repo := branchRepo(t, nil)
	git(t, repo, "remote", "set-url", "origin", "https://github.com/fork-owner/example.git")
	git(t, repo, "remote", "add", "upstream", "https://github.com/"+repoPR+".git")
	commit(t, repo, nil, "feat(app): update upstream\n\nPlan #999")
	git(t, repo, "update-ref", "refs/remotes/upstream/"+base, "HEAD")
	pr := branchAnswer(t, repo, "test(app): add one file", prPR(11))
	pr.Match = []string{"head=" + url.QueryEscape("fork-owner:work")}
	pr.Stdout = strings.ReplaceAll(pr.Stdout, `"source":"`+repoPR+`"`, `"source":"fork-owner/example"`)
	pr.Stdout = strings.ReplaceAll(pr.Stdout, `"base":"main"`, `"base":"`+base+`"`)
	commit(t, repo, nil, "feat(app): add one file\n\nPlan #12")
	return repo, pr
}

// branchReadOnce proves selection does not append a second metadata run.
func branchReadOnce(t *testing.T, rp replays) {
	t.Helper()
	count := 0
	for _, call := range rp.calls("gh") {
		if strings.Contains(call, "pulls?state=open") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("want one branch discovery, got %d: %q", count, rp.calls("gh"))
	}
}

// metadataPush runs real git hooks against a bare push destination and saved PR.
func metadataPushRepo(t *testing.T, title string) (string, []string, replays) {
	t.Helper()
	return stalledPushRepo(t, title, 0, 0)
}

// stalledPushRepo is metadataPushRepo with GitHub stalling planWait seconds
// before it answers the gate's Plan read. With otherWait set, branch other
// holds a docs commit whose PR discovery stalls otherWait seconds.
func stalledPushRepo(t *testing.T, title string, planWait, otherWait int) (string, []string, replays) {
	t.Helper()
	repo := branchRepo(t, nil)
	pr := branchAnswer(t, repo, title, prPR(11))
	plan := planView(t, 11, planBody("- [x] Write the tests."), []string{"Plan"}, "I approve")
	plan.Wait = planWait
	saved := branchGitHub(pr, issuePR(t, 11, planBody("- [x] Write the tests."), "Plan"), plan)
	if otherWait > 0 {
		other := pushedAnswer(t, repo, "other", "docs(readme): touch nothing", prPR(11), 2)
		other.Wait = otherWait
		saved = append(saved, other)
		git(t, repo, "switch", "-q", "-c", "other")
		commit(t, repo, nil, "docs(readme): touch nothing")
		git(t, repo, "switch", "-q", "work")
	}
	rp := standIn(t, map[string][]response{"gh": saved})
	out, code := fitness(t, repo, nil, "init")
	sees(t, out, code, 0, "installed hooks")
	env := pushEnv(installerEnv(t), rp)
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "set-url", "--push", "origin", bare)
	out, code = user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add one file")
	sees(t, out, code, 0, "feat(app): add one file")
	return repo, env, rp
}

func metadataPush(t *testing.T, title string) (string, int, replays) {
	t.Helper()
	repo, env, rp := metadataPushRepo(t, title)
	out, code := user(t, repo, env, "push", "origin", "HEAD:work")
	return out, code, rp
}

func checkoutMain(t *testing.T, repo string) {
	t.Helper()
	git(t, repo, "checkout", "-b", "main", "origin/main")
}

func pushedAnswer(t *testing.T, repo, branch, title, body string, number int) response {
	t.Helper()
	pr := branchAnswer(t, repo, title, body)
	var data map[string]any
	mustDo(t, json.Unmarshal([]byte(pr.Stdout), &data))
	data["head"], data["number"] = branch, number
	raw, err := json.Marshal(data)
	mustDo(t, err)
	pr.Stdout = string(raw)
	pr.Match = []string{"head=" + url.QueryEscape("may-journal:"+branch)}
	return pr
}

// pushedGitHub supplies approval and issue reads alongside the actual pushed
// PRs. Plan #11 has body plan in both the gate's read and the issue read.
func pushedGitHub(t *testing.T, plan string, prs ...response) []response {
	t.Helper()
	saved := []response{{Match: []string{"repo view --json nameWithOwner"}, Stdout: repoPR + "\n"}}
	saved = append(saved, prs...)
	return append(saved, planView(t, 11, plan, []string{"Plan"}, "I approve"), issuePR(t, 11, plan, "Plan"))
}

// hpShims returns the three hook shims for folder dir, each running the
// installer the way the org's synced hooks do.
func hpShims(dir string) map[string]string {
	files := map[string]string{}
	for _, name := range []string{"commit-msg", "pre-commit", "pre-push"} {
		files[dir+"/"+name] = "#!/bin/sh\nexec \"$FITNESS_INSTALL_DIR/fitness-install\" --version latest -- hook " + name + " \"$@\"\n"
	}
	return files
}

// hpStale copies happyRepo with hook shims committed and core.hooksPath left
// at stale, as a clone set up before the folder moved.
func hpStale(t *testing.T, stale string, shipped ...string) (string, []string) {
	t.Helper()
	files := map[string]string{}
	for _, dir := range shipped {
		for name, body := range hpShims(dir) {
			files[name] = body
		}
	}
	repo := example(t, "happyRepo", files)
	git(t, repo, "config", "core.hooksPath", stale)
	return repo, installerEnv(t)
}

// hpPath reads the clone's core.hooksPath.
func hpPath(t *testing.T, repo string) string {
	t.Helper()
	out, _ := exec.Command("git", "-C", repo, "config", "core.hooksPath").Output()
	return strings.TrimSpace(string(out))
}

// hpWants fails unless the clone's core.hooksPath is want.
func hpWants(t *testing.T, repo, want string) {
	t.Helper()
	if got := hpPath(t, repo); got != want {
		t.Errorf("core.hooksPath = %q, want %q", got, want)
	}
}

// misspelledNote is a note cspell fails.
const misspelledNote = "A mispeled wrd.\n"

// hookEvent pipes one Claude Code hook event, as JSON, into the installed
// `fitness claude-hook` run from repo with a cache folder of its own. It
// asserts stdout holds only the JSON Claude Code parses, and returns the
// output and exit code.
func hookEvent(t *testing.T, repo string, event map[string]any) (string, int) {
	t.Helper()
	return hookEventIn(t, repo, t.TempDir(), event)
}

// hookEventIn is hookEvent with the fitness cache folder cache, where the
// hook writes its log.
func hookEventIn(t *testing.T, repo, cache string, event map[string]any) (string, int) {
	t.Helper()
	out, code, _ := timedHook(t, filepath.Join(installOnly(t), "fitness"), repo, cache, event)
	return out, code
}

// timedHook runs the hook event through runner and also returns how long
// the hook alone took.
func timedHook(t *testing.T, runner, repo, cache string, event map[string]any) (string, int, time.Duration) {
	t.Helper()
	event["cwd"] = repo
	raw, err := json.Marshal(event)
	mustDo(t, err)
	cmd := exec.Command(runner, "claude-hook")
	cmd.Dir, cmd.Env, cmd.Stdin = repo, hookEnv(t, cache), strings.NewReader(string(raw))
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	_ = cmd.Run()
	took := time.Since(start)
	onlyJSON(t, stdout.String())
	return stdout.String() + stderr.String(), cmd.ProcessState.ExitCode(), took
}

// hookCopies starts two copies of event at once, sharing a cache
// folder, as Claude Code starts the skill's and the settings' copies.
func hookCopies(t *testing.T, repo string, event map[string]any) (outs [2]string, codes [2]int) {
	t.Helper()
	runner := filepath.Join(installOnly(t), "fitness")
	event["cwd"] = repo
	raw, err := json.Marshal(event)
	mustDo(t, err)
	cache := t.TempDir()
	var cmds [2]*exec.Cmd
	var stdout, stderr [2]strings.Builder
	for i := range cmds {
		cmds[i] = exec.Command(runner, "claude-hook")
		cmds[i].Dir, cmds[i].Env, cmds[i].Stdin = repo, hookEnv(t, cache), strings.NewReader(string(raw))
		cmds[i].Stdout, cmds[i].Stderr = &stdout[i], &stderr[i]
	}
	for _, cmd := range cmds {
		mustDo(t, cmd.Start())
	}
	for i, cmd := range cmds {
		_ = cmd.Wait()
		onlyJSON(t, stdout[i].String())
		outs[i], codes[i] = stdout[i].String()+stderr[i].String(), cmd.ProcessState.ExitCode()
	}
	return outs, codes
}

// hookEnv is the user's environment with the fitness cache folder cache.
func hookEnv(t *testing.T, cache string) []string {
	t.Helper()
	return append(userEnv(), "FITNESS_CACHE_DIR="+cache)
}

// onlyJSON asserts out is empty or one JSON object and nothing else.
func onlyJSON(t *testing.T, out string) {
	t.Helper()
	var verdict map[string]any
	if out != "" && json.Unmarshal([]byte(out), &verdict) != nil {
		t.Errorf("stdout must hold only the hook's JSON:\n%s", out)
	}
}

// editEvent is the PostToolUse event for a Write of file in repo.
func editEvent(repo, file string) map[string]any {
	return map[string]any{"hook_event_name": "PostToolUse", "tool_name": "Write", "tool_input": map[string]any{"file_path": filepath.Join(repo, file)}}
}

// bashEvent is the PreToolUse event for a Bash call running command.
func bashEvent(command string) map[string]any {
	return map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Bash", "tool_input": map[string]any{"command": command}}
}

// stopEvent is the Stop event, with stop_hook_active as active.
func stopEvent(active bool) map[string]any {
	return map[string]any{"hook_event_name": "Stop", "stop_hook_active": active}
}

// hookEvents are the events the installer and the fitness skill register,
// in order.
var hookEvents = []string{"PostToolUse", "PreToolUse", "Stop", "SubagentStop", "SessionStart", "SessionEnd"}

// silentPass asserts the hook let the event pass without a word.
func silentPass(t *testing.T, out string, code int) {
	t.Helper()
	if code != 0 || out != "" {
		t.Errorf("want a silent pass, got exit %d:\n%s", code, out)
	}
}

// claudeSettings runs `fitness-install --claude-settings` on file and
// returns the settings it leaves.
func claudeSettings(t *testing.T, file string) map[string]any {
	t.Helper()
	out, code := fiRun(t, t.TempDir(), nil, "--claude-settings", file)
	sees(t, out, code, 0, "fitness: Claude hooks in "+file+" run ")
	var settings map[string]any
	mustDo(t, json.Unmarshal([]byte(fiFile(t, file)), &settings))
	return settings
}

// fitnessHandlers lists, per event, the commands of the handlers that run
// `claude-hook`, and returns every other command too.
func fitnessHandlers(settings map[string]any) (ours map[string][]string, others []string) {
	ours = map[string][]string{}
	hooks, _ := settings["hooks"].(map[string]any)
	for event, groups := range hooks {
		for _, g := range groups.([]any) {
			for _, h := range g.(map[string]any)["hooks"].([]any) {
				command := h.(map[string]any)["command"].(string)
				if strings.HasSuffix(command, " claude-hook") {
					ours[event] = append(ours[event], command)
				} else {
					others = append(others, command)
				}
			}
		}
	}
	return ours, others
}

// keptMode asserts file still has mode.
func keptMode(t *testing.T, file string, mode os.FileMode) {
	t.Helper()
	if info, err := os.Stat(file); err != nil || info.Mode().Perm() != mode {
		t.Errorf("%s must keep its mode %o: %v", file, mode, err)
	}
}

// skillDeclares asserts the fitness skill's front matter ends with the hooks
// the installer wrote into settings: the same events, matchers and timeouts.
func skillDeclares(t *testing.T, settings map[string]any) {
	t.Helper()
	hooks := settings["hooks"].(map[string]any)
	want := "hooks:\n"
	for _, event := range hookEvents {
		want += "  " + event + ":\n" + declared(hooks[event].([]any))
	}
	skill := fiFile(t, filepath.Join("..", "..", "skills", "fitness", "SKILL.md"))
	if !strings.Contains(skill, "\n"+want+"---\n") {
		t.Errorf("the fitness skill front matter must end with these hooks:\n%s\ngot:\n%s", want, skill)
	}
}

// declared is the front matter YAML for the last group of an event, which
// the installer appends.
func declared(groups []any) string {
	group := groups[len(groups)-1].(map[string]any)
	handler := group["hooks"].([]any)[0].(map[string]any)
	yaml := "    - hooks:\n"
	if matcher, ok := group["matcher"].(string); ok {
		yaml = "    - matcher: \"" + matcher + "\"\n      hooks:\n"
	}
	return yaml + fmt.Sprintf("        - type: command\n          command: fitness claude-hook\n          timeout: %v\n", handler["timeout"])
}

// installedOnce asserts event holds one Fitness hook, run by the installed
// fitness.
func installedOnce(t *testing.T, event string, commands []string) {
	t.Helper()
	if len(commands) != 1 {
		t.Fatalf("%s must hold the Fitness hook once, got %v", event, commands)
	}
	runner := strings.TrimSuffix(commands[0], " claude-hook")
	if info, err := os.Stat(runner); err != nil || filepath.Base(runner) != "fitness" || info.IsDir() {
		t.Errorf("%s runs %q, not the installed fitness", event, runner)
	}
}

// startEvent is the SessionStart event from source.
func startEvent(source string) map[string]any {
	return map[string]any{"hook_event_name": "SessionStart", "source": source}
}

// editTranscript is a subagent transcript whose Write calls name files.
func editTranscript(t *testing.T, files ...string) string {
	t.Helper()
	transcript := filepath.Join(t.TempDir(), "agent.jsonl")
	lines := `{"type":"user","message":{"content":"Fix the notes."}}` + "\n"
	for _, file := range files {
		lines += fmt.Sprintf(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":%q}}]}}`, file) + "\n"
	}
	mustDo(t, os.WriteFile(transcript, []byte(lines), 0o600))
	return transcript
}

// subagentStop is the SubagentStop event of agent in session s1. Its last
// message differs per try, as each real stop's does.
func subagentStop(agent string, try int, active bool, transcript string) map[string]any {
	return map[string]any{"hook_event_name": "SubagentStop", "session_id": "s1", "agent_id": agent, "stop_hook_active": active,
		"agent_transcript_path": transcript, "last_assistant_message": fmt.Sprintf("Done, try %d.", try)}
}

// onBranch puts repo on a feature branch off origin/main, which points at
// its current commit.
func onBranch(t *testing.T, repo string) {
	t.Helper()
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	git(t, repo, "checkout", "-q", "-b", "feature")
}

// hookCounts are a log line's or total's counts.
type hookCounts struct {
	Runs, Blocks int
	Failed       int `json:"failedChecks"`
	Lines        int `json:"findingLines"`
}

// loggedEditsAndTotal reports whether lines are a red and a clean edit and
// then a SessionEnd total of both, one blocked with one finding.
func loggedEditsAndTotal(lines []string) bool {
	var red, clean hookCounts
	var total struct {
		Event string
		Hooks map[string]hookCounts
	}
	err := errors.Join(json.Unmarshal([]byte(lines[0]), &red), json.Unmarshal([]byte(lines[1]), &clean), json.Unmarshal([]byte(lines[2]), &total))
	edits := hookCounts{Runs: 2, Blocks: 1, Failed: 1, Lines: 1}
	return err == nil && red == hookCounts{Failed: 1, Lines: 1} && clean == hookCounts{} && total.Event == "SessionEnd" && total.Hooks["PostToolUse"] == edits
}

// failingGoTest is a Go module whose one test fails wherever go-test runs.
var failingGoTest = map[string]string{
	".fitnessrc.json": `{"policy": "external", "disabledChecks": ["requirements"]}` + "\n",
	"go.mod":          "module example.com/hooked\n\ngo 1.22\n",
	"hooked.go":       "// Package hooked holds one constant.\npackage hooked\n\n// Answer is the answer.\nconst Answer = 42\n",
	"hooked_test.go":  "package hooked\n\nimport \"testing\"\n\nfunc TestAnswer(t *testing.T) { t.Fatal(\"go-test ran\") }\n",
}

// hookTimeout is the timeout, in seconds, of event's first hook in settings.
func hookTimeout(hooks map[string]any, event string) any {
	group := hooks[event].([]any)[0].(map[string]any)
	return group["hooks"].([]any)[0].(map[string]any)["timeout"]
}

// claudeStop is Claude's Stop event in session s1 with its transcript, never
// marked active. Its last message differs per try, as each real stop's does.
func claudeStop(try int, transcript string) map[string]any {
	return map[string]any{"hook_event_name": "Stop", "session_id": "s1", "transcript_path": transcript, "stop_hook_active": false,
		"last_assistant_message": fmt.Sprintf("Done, try %d.", try)}
}

// addUserLine appends a user line of prompt to the transcript, as Claude
// Code writes a prompt or a Stop hook's feedback.
func addUserLine(t *testing.T, transcript, prompt, text string) {
	t.Helper()
	f, err := os.OpenFile(transcript, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	mustDo(t, err)
	defer f.Close()
	_, err = fmt.Fprintf(f, `{"type":"user","promptId":%q,"uuid":"%s-%d","message":{"role":"user","content":%q}}`+"\n", prompt, prompt, len(text), text)
	mustDo(t, err)
}

// slowChecks are local checks, four per processor, that each pass after
// four seconds, so together they outlast the start hook's deadline.
func slowChecks() map[string]string {
	files := map[string]string{".fitnessrc.json": `{"localChecks": "checks"}` + "\n"}
	for i := range 4 * runtime.NumCPU() {
		name := fmt.Sprintf("slow-%d", i)
		files["checks/fitness-check-"+name] = "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\":\"" + name +
			"\",\"scope\":\"files\"}'; exit 0; }\nsleep 4\necho '" + passing + "'\n"
	}
	return files
}

// unpinnedPrePush is the pre-push hook as synced before it set
// FITNESS_WITH_CHECKS, which hands the engine no checks.
const unpinnedPrePush = "#!/bin/sh\nexec \"${FITNESS_INSTALL_DIR:-$HOME/.local/share/fitness}/fitness-install\" --version latest -- hook pre-push\n"

// pushPrePush pushes to a PR with a bold body from a repo pinning every check,
// none installed. It runs hook, else the `fitness init` one.
func pushPrePush(t *testing.T, hook string) (string, int) {
	t.Helper()
	repo := branchRepo(t, map[string]string{".fitnessrc.json": `{"checkVersions": {"*": "v1"}}`})
	const title = "test(app): add one file"
	pr := branchAnswer(t, repo, title, prPR(11)+"\nSome **bold** text.\n")
	rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issuePR(t, 11, planBody("- [x] Write the tests."), "Plan"),
		planView(t, 11, planBody("- [x] Write the tests."), []string{"Plan"}, "I approve"))})
	out, code := fitness(t, repo, nil, "init")
	sees(t, out, code, 0, "installed hooks")
	if hook != "" {
		mustDo(t, os.WriteFile(filepath.Join(repo, ".githooks", "pre-push"), []byte(hook), 0o755))
	}
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "set-url", "--push", "origin", bare)
	env := append(pushEnv(installerEnv(t), rp), "FITNESS_CHECKS_DIR=")
	out, code = user(t, repo, env, "commit", "--allow-empty", "-m", title)
	sees(t, out, code, 0, title)
	return user(t, repo, env, "push", "origin", "work")
}
