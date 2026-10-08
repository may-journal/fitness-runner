package requirements

import "testing"

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

func Test0011_1(t *testing.T) {
	repo, env := linked(t)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): again\n\nPlan #218")
	sees(t, out, code, 1, `#218 is already linked by commit`, `("chore(app): start")`)
}

func Test0011_2(t *testing.T) {
	repo, env := linked(t)
	out, code := user(t, repo, env, "commit", "--amend", "--allow-empty", "-m", "chore(app): start over\n\nPlan #218")
	sees(t, out, code, 0, "chore(app): start over")
}
