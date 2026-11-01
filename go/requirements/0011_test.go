package requirements

import (
	"testing"
	"time"
)

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
	t.Parallel()
	repo, env := linked(t)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): again\n\nPlan #218")
	sees(t, out, code, 1, `#218 is already linked by commit`, `("chore(app): start")`)
}

func Test0011_2(t *testing.T) {
	t.Parallel()
	repo, env := linked(t)
	out, code := user(t, repo, env, "commit", "--amend", "--allow-empty", "-m", "chore(app): start over\n\nPlan #218")
	sees(t, out, code, 0, "chore(app): start over")
}

func Test0011_3(t *testing.T) {
	t.Parallel()
	repo, env := linked(t)
	user(t, repo, env, "push", "-q", "origin", "HEAD:epic")
	user(t, repo, env, "switch", "-q", "-c", "child")
	out, code := user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "chore(app): child\n\nPlan #218")
	sees(t, out, code, 0)
	out, code = user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): again\n\nPlan #218")
	sees(t, out, code, 1, `#218 is already linked by commit`, `("chore(app): child")`)
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

func Test0011_4(t *testing.T) {
	t.Parallel()
	repo, env := linked(t)
	recordedNow(t, env)
	out, code := silentCommit(t, repo, env, "chore(app): again\n\nPlan #218")
	sees(t, out, code, 1, `#218 is already linked by commit`, offlineNote, "so fitness skipped the open PR read without connecting")
	out, code = silentCommit(t, repo, env, "chore(app): next\n\nPlan #219")
	sees(t, out, code, 0, offlineNote, "chore(app): next")
}
