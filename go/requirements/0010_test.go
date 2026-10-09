package requirements

import "testing"

// planned is a hooked repo whose pre-push reads Plans from fitness-runner's
// real issues: #218 is approved, #151 is a Plan nobody approved, and #196 is
// not a Plan. It pushes main, so later pushes start from a shared base.
func planned(t *testing.T) (string, []string) {
	t.Helper()
	offMachine(t)
	repo, env := hooked(t, nil)
	env = append(env, "GH_REPO=may-journal/fitness-runner")
	remote(t, repo)
	if out, code := user(t, repo, env, "push", "-q", "origin", "HEAD:main"); code != 0 {
		t.Fatalf("pushing main: %s", out)
	}
	return repo, env
}

// pushFeature commits a feature naming the Plan trailer and pushes it.
func pushFeature(t *testing.T, trailer string) (string, int) {
	t.Helper()
	repo, env := planned(t)
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): add a feature\n\n"+trailer)
	return user(t, repo, env, "push", "origin", "HEAD:main")
}

func Test0010_1(t *testing.T) {
	t.Parallel()
	out, code := pushFeature(t, "Plan #218")
	sees(t, out, code, 0, "HEAD -> main")
}

func Test0010_2(t *testing.T) {
	t.Parallel()
	out, code := pushFeature(t, "Plan #196")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (196)")
}

func Test0010_3(t *testing.T) {
	t.Parallel()
	out, code := pushFeature(t, "Plan #151")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (151)")
}

func Test0010_4(t *testing.T) {
	t.Parallel()
	repo, env := planned(t)
	user(t, repo, env, "switch", "-q", "-c", "topic")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): start\n\nPlan #218")
	if out, code := user(t, repo, env, "push", "-q", "origin", "topic"); code != 0 {
		t.Fatalf("first push: %s", out)
	}
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "fix(app): follow up")
	out, code := user(t, repo, env, "push", "origin", "topic")
	sees(t, out, code, 0, "topic -> topic")
}

func Test0010_5(t *testing.T) {
	t.Parallel()
	repo, env := planned(t)
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
	repo, env := planned(t)
	user(t, repo, env, "push", "-q", "origin", "HEAD:topic")
	out, code := user(t, repo, env, "push", "origin", "--delete", "topic")
	sees(t, out, code, 0, "[deleted]", "topic")
}
