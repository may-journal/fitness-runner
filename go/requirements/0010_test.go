package requirements

import (
	"os"
	"strings"
	"testing"
)

// pushPlans are GitHub's saved answers about the issues a trailer may name:
// #218 is an approved Plan, #151 is a Plan nobody approved, and #196 is a bug.
var pushPlans = []response{
	{Match: []string{"issue view 218 ", "labels"}, Stdout: "Plan\n"},
	{Match: []string{"issue view 218 ", "comments"}, Stdout: "Looks good.\nApproved, go ahead.\n"},
	{Match: []string{"issue view 151 ", "labels"}, Stdout: "Plan\nenhancement\n"},
	{Match: []string{"issue view 151 ", "comments"}, Stdout: "Could this reuse the existing parser?\n"},
	{Match: []string{"issue view 196 ", "labels"}, Stdout: "bug\n"},
	{Match: []string{"issue view 196 ", "comments"}, Stdout: "Approved.\n"},
}

// pushRepo is a hooked repo whose pre-push asks the gh stand-in about Plans.
// It pushes main, so later pushes start from a shared base.
func pushRepo(t *testing.T) (string, []string, replays) {
	t.Helper()
	repo, hookEnv := hooked(t, nil)
	rp := standIn(t, map[string][]response{"gh": pushPlans})
	env := pushEnv(hookEnv, rp)
	remote(t, repo)
	if out, code := user(t, repo, env, "push", "-q", "origin", "HEAD:main"); code != 0 {
		t.Fatalf("pushing main: %s", out)
	}
	return repo, env, rp
}

// pushEnv puts the stand-ins on PATH ahead of the installed fitness.
func pushEnv(hookEnv []string, rp replays) []string {
	sep := string(os.PathListSeparator)
	fitnessPath := strings.TrimPrefix(hookEnv[0], "PATH=")
	var env []string
	for _, kv := range rp.env {
		if path, ok := strings.CutPrefix(kv, "PATH="); ok {
			kv = "PATH=" + strings.SplitN(path, sep, 2)[0] + sep + fitnessPath
		}
		env = append(env, kv)
	}
	return env
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

func Test0010_1(t *testing.T) {
	t.Parallel()
	out, code, rp := pushFeature(t, "Plan #218")
	sees(t, out, code, 0, "HEAD -> main")
	pushAsked(t, rp, "218", "comments")
}

func Test0010_2(t *testing.T) {
	t.Parallel()
	out, code, rp := pushFeature(t, "Plan #196")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (196)")
	pushAsked(t, rp, "196", "labels")
}

func Test0010_3(t *testing.T) {
	t.Parallel()
	out, code, rp := pushFeature(t, "Plan #151")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (151)")
	pushAsked(t, rp, "151", "comments")
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

func Test0010_5(t *testing.T) {
	t.Parallel()
	repo, env, _ := pushRepo(t)
	user(t, repo, env, "switch", "-q", "-c", "topic")
	user(t, repo, env, "switch", "-q", "main")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): merged by review (#12)")
	user(t, repo, env, "push", "-q", "--no-verify", "origin", "HEAD:main")
	user(t, repo, env, "switch", "-q", "topic")
	user(t, repo, env, "merge", "-q", "--no-edit", "main")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "chore(app): tidy up")
	out, code := user(t, repo, env, "push", "origin", "topic")
	sees(t, out, code, 0, "topic -> topic")
}

func Test0010_6(t *testing.T) {
	t.Parallel()
	repo, env, _ := pushRepo(t)
	user(t, repo, env, "push", "-q", "origin", "HEAD:topic")
	out, code := user(t, repo, env, "push", "origin", "--delete", "topic")
	sees(t, out, code, 0, "[deleted]", "topic")
}
