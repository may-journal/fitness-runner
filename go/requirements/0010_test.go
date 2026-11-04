package requirements

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

// Comment authors on the saved Plans: the repo's owner, a passer-by, an org
// member, and a reviewer listed in planApprovers.
var (
	owner    = func(body string) comment { return comment{Body: body, Author: "example-owner", Association: "OWNER"} }
	passerby = func(body string) comment { return comment{Body: body, Author: "passerby", Association: "NONE"} }
	member   = func(body string) comment { return comment{Body: body, Author: "org-member", Association: "MEMBER"} }
	reviewer = func(body string) comment { return comment{Body: body, Author: "reviewer-bot", Association: "NONE"} }
)

// pushPlans are saved GitHub answers about issues a PR may close: #218
// owner-approved, #196 a bug, #151 passer-by, #307 reviewer, #308 member.
func pushPlans(t *testing.T) []response {
	t.Helper()
	plan, labels := planBody("- [ ] Ship it."), []string{"Plan"}
	return []response{
		planViewBy(t, 218, plan, labels, passerby("Looks good."), owner("I approve this Plan.")),
		planViewBy(t, 151, plan, []string{"Plan", "enhancement"}, owner("This is not approved; it needs work."), passerby("I approve.")),
		planView(t, 196, "The parser drops the last line.", []string{"bug"}, "I approve."),
		planViewBy(t, 307, plan, labels, reviewer("I approve.")),
		planViewBy(t, 308, plan, labels, member("i approve, go ahead")),
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

// githubRepo is a hooked repo with config rc whose origin is repo0014 on
// GitHub, pushed to a bare folder instead.
func githubRepo(t *testing.T, rc string) (string, []string) {
	t.Helper()
	repo, hookEnv := hooked(t, map[string]string{".fitnessrc.json": rc})
	git(t, repo, "remote", "add", "origin", "https://github.com/"+repo0014+".git")
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
	saved := []response{{Match: []string{"repo view --json nameWithOwner"}, Stdout: repo0014 + "\n"}}
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

func Test0010_11(t *testing.T) {
	t.Parallel()
	repo, env, rp := pushRepo(t, gateOnly, "Closes #218\n\nFixes #218")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): start")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): continue")
	user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "fix(app): follow up")
	out, code := user(t, repo, env, "push", "origin", "topic")
	sees(t, out, code, 0, "topic -> topic")
	planReadOnce(t, rp, "218")
	silentPushRefused(t)
}

func Test0010_1(t *testing.T) {
	t.Parallel()
	out, code, rp := pushClosing(t, "Closes #218")
	sees(t, out, code, 0, "topic -> topic")
	pushAsked(t, rp, "218", "comments")
	out, code, _ = pushClosing(t, "Fixes #308")
	sees(t, out, code, 0, "topic -> topic")
	out, code, _ = pushFeature(t, listed, "Resolves #307", "feat(app): add a feature")
	sees(t, out, code, 0, "topic -> topic")
}

func Test0010_4(t *testing.T) {
	t.Parallel()
	out, code, rp := pushClosing(t, "")
	sees(t, out, code, 0, "pre-push: info: topic has no open PR yet, so its PR check gates the Plan.", "topic -> topic")
	if !rp.called("gh", "head=may-journal%3Atopic") || rp.called("gh", "issue view") {
		t.Errorf("want one PR lookup for topic and no Plan read; gh calls: %q", rp.calls("gh"))
	}
}

func Test0010_10(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	remote(t, repo)
	user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): tidy up")
	out, code := user(t, repo, env, "push", "origin", "HEAD:main")
	sees(t, out, code, 0, "HEAD -> main")
	repo, env, _ = pushRepo(t, gateOnly, "")
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
	repo, env, _ := pushRepo(t, gateOnly, "")
	user(t, repo, env, "push", "-q", "origin", "HEAD:topic")
	out, code := user(t, repo, env, "push", "origin", "--delete", "topic")
	sees(t, out, code, 0, "[deleted]", "topic")
}

// silentGitHub is the environment of the real gh, with a token, behind a
// silent proxy.
func silentGitHub(t *testing.T) []string {
	t.Helper()
	if _, err := exec.LookPath("gh"); err != nil {
		t.Fatalf("this test runs the real gh: %v", err)
	}
	return []string{"HTTPS_PROXY=" + silentListener(t), "NO_PROXY=", "no_proxy=", "GH_TOKEN=unused", "GH_HOST=github.com",
		"GH_REPO=" + repo0014, "GH_CONFIG_DIR=" + t.TempDir(), "GH_NO_UPDATE_NOTIFIER=1"}
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

func Test0010_12(t *testing.T) {
	t.Parallel()
	off := `{"policy": "external", "disabledChecks": ["plan-trailer", "local-branch"]}` + "\n"
	repo, env, rp := pushRepo(t, off, "Refs #218")
	user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add a feature")
	out, code := user(t, repo, env, "push", "origin", "topic")
	sees(t, out, code, 0, "topic -> topic")
	if rp.called("gh", "head=may-journal%3Atopic") {
		t.Errorf("the gate ran with plan-trailer off; gh calls: %q", rp.calls("gh"))
	}
}

// orgPlanView is GitHub's saved answer about issue n of repo, read with -R.
func orgPlanView(t *testing.T, repo string, n int, labels []string, comments ...string) response {
	t.Helper()
	r := planView(t, n, planBody("- [ ] Ship it."), labels, comments...)
	r.Stdout = strings.Replace(r.Stdout, repo0014, repo, 1)
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

func Test0010_13(t *testing.T) {
	t.Parallel()
	out, code, rp := pushClosing(t, "Closes May-Journal/.github#144")
	sees(t, out, code, 0, "topic -> topic")
	if n := rp.count("gh", "issue view 144 ", "-R May-Journal/.github"); n != 1 {
		t.Errorf("want one read of May-Journal/.github#144, got %d; gh calls: %q", n, rp.calls("gh"))
	}
	out, code, rp = pushClosing(t, "Fixes https://github.com/may-journal/.github/issues/144")
	sees(t, out, code, 0, "topic -> topic")
	pushAsked(t, rp, "144", "-R may-journal/.github")
}

func Test0010_14(t *testing.T) {
	t.Parallel()
	out, code, _ := pushFeature(t, gateOnly, "Refs #218, and `Closes #218` in code.", "feat(app): add a feature\n\nPlan #218")
	sees(t, out, code, 1, "pre-push: https://github.com/"+repo0014+"/pull/1 closes no Plan.",
		"The PR body must close the approved Plan: 'Closes #NN', or 'Closes <owner>/<repo>#NN' for a Plan in another repo")
	out, code, rp := pushClosing(t, "Closes #196")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (196)")
	pushAsked(t, rp, "196", "labels")
	out, code, rp = pushClosing(t, "Closes #151")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (151)",
		`A Plan needs a comment starting "I approve" from the repo's owner or an org member (planApprovers is unset).`)
	pushAsked(t, rp, "151", "comments")
	out, code, _ = pushClosing(t, "Closes #307")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (307)")
	out, code, _ = pushFeature(t, listed, "Closes #218", "feat(app): add a feature")
	sees(t, out, code, 1, "pre-push: no approved Plan Issue among (218)",
		`A Plan needs a comment starting "I approve" from one of planApprovers: Reviewer-Bot.`)
	out, code, rp = pushClosing(t, "Closes may-journal/.github#141")
	sees(t, out, code, 1, "no approved Plan Issue among (may-journal/.github#141)")
	pushAsked(t, rp, "141", "-R may-journal/.github")
	out, code, rp = pushClosing(t, "Closes other-org/.github#7")
	sees(t, out, code, 1, "no approved Plan Issue among (other-org/.github#7)", `Plan other-org/.github#7 is refused: only a Plan owned by "may-journal" counts here.`)
	if rp.called("gh", "issue view 7 ") {
		t.Errorf("pre-push read another owner's Plan; gh calls: %q", rp.calls("gh"))
	}
}
