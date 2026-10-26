package requirements

import (
	"slices"
	"strings"
	"testing"
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

func Test0010_9(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add a feature")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 1, "pre-push: no 'Plan #NN' trailer")
	out, code, rp := pushFeature(t, "Plan #196")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (196)")
	pushAsked(t, rp, "196", "labels")
	out, code, rp = pushFeature(t, "Plan #151")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (151)")
	pushAsked(t, rp, "151", "comments")
}

func Test0010_12(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["plan-trailer", "local-branch"]}` + "\n"})
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add a feature")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 0)
}
