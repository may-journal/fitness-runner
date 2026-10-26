package requirements

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

// pushPlans are GitHub's saved answers about the issues a trailer may name:
// #218 is an approved Plan, #151 is a Plan nobody approved, and #196 is a bug.
func pushPlans(t *testing.T) []response {
	t.Helper()
	return []response{
		planView(t, 218, planBody("- [ ] Ship it."), []string{"Plan"}, "Looks good.", "Approved, go ahead."),
		planView(t, 151, planBody("- [ ] Ship it."), []string{"Plan", "enhancement"}, "Could this reuse the existing parser?"),
		planView(t, 196, "The parser drops the last line.", []string{"bug"}, "Approved."),
	}
}

// pushRepo is a hooked repo whose pre-push asks the gh stand-in about Plans.
// It pushes main, so later pushes start from a shared base.
func pushRepo(t *testing.T) (string, []string, replays) {
	t.Helper()
	repo, hookEnv := hooked(t, nil)
	rp := standIn(t, map[string][]response{"gh": pushPlans(t)})
	env := pushEnv(hookEnv, rp)
	remote(t, repo)
	if out, code := user(t, repo, env, "push", "-q", "origin", "HEAD:main"); code != 0 {
		t.Fatalf("pushing main: %s", out)
	}
	return repo, env, rp
}

// pushEnv adds the stand-ins, first on PATH, to the environment that already holds the installer.
func pushEnv(hookEnv []string, rp replays) []string {
	return slices.Concat(hookEnv, rp.env)
}

// pushFeature commits a feature naming the Plan trailer and pushes it.
func pushFeature(t *testing.T, trailer string) (string, int, replays) {
	t.Helper()
	repo, env, rp := pushRepo(t)
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): add a feature\n\n"+trailer)
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	return out, code, rp
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

func Test0010_11(t *testing.T) {
	t.Parallel()
	repo, env, rp := pushRepo(t)
	user(t, repo, env, "switch", "-q", "-c", "topic")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): start\n\nPlan #218")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): continue")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "fix(app): follow up")
	out, code := user(t, repo, env, "push", "origin", "topic")
	sees(t, out, code, 0, "topic -> topic")
	planReadOnce(t, rp, "218")
	silentPushRefused(t)
}

func Test0010_1(t *testing.T) {
	t.Parallel()
	out, code, rp := pushFeature(t, "Plan #218")
	sees(t, out, code, 0, "HEAD -> main")
	pushAsked(t, rp, "218", "comments")
}

func Test0010_4(t *testing.T) {
	t.Parallel()
	repo, env, rp := pushRepo(t)
	user(t, repo, env, "switch", "-q", "-c", "topic")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): start\n\nPlan #218")
	if out, code := user(t, repo, env, "push", "-q", "origin", "topic"); code != 0 {
		t.Fatalf("first push: %s", out)
	}
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "fix(app): follow up")
	out, code := user(t, repo, env, "push", "origin", "topic")
	sees(t, out, code, 0, "topic -> topic")
	pushAsked(t, rp, "218", "comments")
}

func Test0010_10(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): tidy up")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 0, "HEAD -> main")
	repo, env, _ = pushRepo(t)
	user(t, repo, env, "switch", "-q", "-c", "topic")
	user(t, repo, env, "switch", "-q", "main")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): merged by review (#12)")
	user(t, repo, env, "push", "-q", "--no-verify", "origin", "HEAD:main")
	user(t, repo, env, "switch", "-q", "topic")
	user(t, repo, env, "merge", "-q", "--no-edit", "main")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "chore(app): tidy up")
	out, code = user(t, repo, env, "push", "origin", "topic")
	sees(t, out, code, 0, "topic -> topic")
}

func Test0010_6(t *testing.T) {
	t.Parallel()
	repo, env, _ := pushRepo(t)
	user(t, repo, env, "push", "-q", "origin", "HEAD:topic")
	out, code := user(t, repo, env, "push", "origin", "--delete", "topic")
	sees(t, out, code, 0, "[deleted]", "topic")
}

// silentGitHub is the environment of the real gh behind a proxy that
// accepts connections and never answers, with a token so it asks GitHub.
func silentGitHub(t *testing.T) []string {
	t.Helper()
	if _, err := exec.LookPath("gh"); err != nil {
		t.Fatalf("this test runs the real gh: %v", err)
	}
	return []string{"HTTPS_PROXY=" + silentListener(t), "NO_PROXY=", "no_proxy=", "GH_TOKEN=unused", "GH_HOST=github.com",
		"GH_REPO=" + repo0014, "GH_CONFIG_DIR=" + t.TempDir(), "GH_NO_UPDATE_NOTIFIER=1"}
}

// silentPushRefused pushes a commit naming three Plans behind a GitHub that
// never answers. Three Plans once waited 30 seconds each; now the first
// read to run out ends the push with one line.
func silentPushRefused(t *testing.T) {
	t.Helper()
	repo, hookEnv := hooked(t, nil)
	remote(t, repo)
	online := pushEnv(hookEnv, standIn(t, map[string][]response{"gh": pushPlans(t)}))
	out, code := user(t, repo, online, "commit", "--allow-empty", "-m", "feat(app): add a feature\n\nPlan #218\nPlan #151\nPlan #196")
	sees(t, out, code, 0, "feat(app): add a feature")
	start := time.Now()
	out, code = user(t, repo, append(hookEnv, silentGitHub(t)...), "push", "origin", "HEAD:main")
	sees(t, out, code, 1, "pre-push: github.com did not answer within 30s; check the network or proxy, then push again.")
	if took := time.Since(start); took > 40*time.Second {
		t.Errorf("push took %s, past the 30-second deadline", took)
	}
	if n := strings.Count(out, "pre-push:"); n != 1 {
		t.Errorf("want one pre-push line, got %d:\n%s", n, out)
	}
}

func Test0010_12(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["plan-trailer", "local-branch"]}` + "\n"})
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add a feature")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 0)
}

// orgPlanView is GitHub's saved answer about issue n of repo, read with -R.
func orgPlanView(t *testing.T, repo string, n int, labels []string, comments ...string) response {
	t.Helper()
	r := planView(t, n, planBody("- [ ] Ship it."), labels, comments...)
	r.Stdout = strings.Replace(r.Stdout, repo0014, repo, 1)
	r.Match = append(r.Match, "-R "+repo)
	return r
}

// orgPlans are GitHub's saved answers about Plans in other repos:
// may-journal/.github#144 is approved however its owner is cased,
// may-journal/.github#141 is not, and
// other-org/.github#7 is approved but belongs to another owner.
func orgPlans(t *testing.T) []response {
	t.Helper()
	return []response{
		orgPlanView(t, "may-journal/.github", 144, []string{"Plan"}, "I approve"),
		orgPlanView(t, "May-Journal/.github", 144, []string{"Plan"}, "I approve"),
		orgPlanView(t, "may-journal/.github", 141, []string{"Plan"}, "Could this wait a week?"),
		orgPlanView(t, "other-org/.github", 7, []string{"Plan"}, "I approve"),
	}
}

// orgPushFeature commits a feature naming trailer in a repo whose origin is
// repo0014 on GitHub, pushed to a bare folder, and pushes it. local-branch
// is off, so the push asks GitHub only about Plans.
func orgPushFeature(t *testing.T, trailer string) (string, int, replays) {
	t.Helper()
	repo, hookEnv := hooked(t, map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["local-branch"]}` + "\n"})
	git(t, repo, "remote", "add", "origin", "https://github.com/"+repo0014+".git")
	bare := t.TempDir()
	git(t, bare, "init", "-q", "--bare")
	git(t, repo, "remote", "set-url", "--push", "origin", bare)
	rp := standIn(t, map[string][]response{"gh": orgPlans(t)})
	env := pushEnv(hookEnv, rp)
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): add a feature\n\n"+trailer)
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	return out, code, rp
}

func Test0010_13(t *testing.T) {
	t.Parallel()
	out, code, rp := orgPushFeature(t, "Plan May-Journal/.github#144")
	sees(t, out, code, 0, "HEAD -> main")
	if n := rp.count("gh", "issue view 144 ", "-R May-Journal/.github"); n != 1 {
		t.Errorf("want one read of May-Journal/.github#144, got %d; gh calls: %q", n, rp.calls("gh"))
	}
	out, code, rp = orgPushFeature(t, "Plan https://github.com/may-journal/.github/issues/144")
	sees(t, out, code, 0, "HEAD -> main")
	pushAsked(t, rp, "144", "-R may-journal/.github")
}

func Test0010_14(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add a feature")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 1, "pre-push: no 'Plan #NN' trailer", "'Plan <owner>/<repo>#NN' for a Plan in another repo")
	out, code, rp := pushFeature(t, "Plan #196")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (196)")
	pushAsked(t, rp, "196", "labels")
	out, code, rp = pushFeature(t, "Plan #151")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (151)")
	pushAsked(t, rp, "151", "comments")
	out, code, rp = orgPushFeature(t, "Plan may-journal/.github#141")
	sees(t, out, code, 1, "no approved Plan Issue among (may-journal/.github#141)")
	pushAsked(t, rp, "141", "-R may-journal/.github")
	out, code, rp = orgPushFeature(t, "Plan other-org/.github#7")
	sees(t, out, code, 1, "no approved Plan Issue among (other-org/.github#7)", `Plan other-org/.github#7 is refused: only a Plan owned by "may-journal" counts here.`)
	if rp.called("gh", "issue view 7 ") {
		t.Errorf("pre-push read another owner's Plan; gh calls: %q", rp.calls("gh"))
	}
}
