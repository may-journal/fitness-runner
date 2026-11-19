// cspell:ignore aproved approvd apprvoed dont
// cspell:ignore mispeled

package requirements

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test0005_1(t *testing.T) {
	cases{
		{"a repo with no hooks and no fitness on the PATH", func(t *testing.T) {
			t.Parallel()
			repo, _ := hooked(t, nil)
			out, _ := exec.Command("git", "-C", repo, "config", "core.hooksPath").Output()
			if got := strings.TrimSpace(string(out)); got != ".githooks" {
				t.Errorf("core.hooksPath = %q, want .githooks", got)
			}
			for name, checks := range map[string]string{"commit-msg": "FITNESS_WITH_CHECKS=pinned exec ", "pre-commit": "FITNESS_WITH_CHECKS=pinned exec ", "pre-push": "FITNESS_WITH_CHECKS=pinned exec "} {
				data, _ := os.ReadFile(filepath.Join(repo, ".githooks", name))
				if !strings.Contains(string(data), "\n"+checks+`"${FITNESS_INSTALL_DIR:-$HOME/.local/share/fitness}/fitness-install" --version latest -- hook `+name) {
					t.Errorf(".githooks/%s does not run the installer for hook %s as %q", name, name, checks)
				}
			}
		}},
		{"a repo that already has a pre-commit hook", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			mine := "#!/bin/sh\necho my own hook\n"
			write(t, repo, map[string]string{".git/hooks/pre-commit": mine})
			cmd := exec.Command(installer(t), "--install-hook")
			cmd.Dir, cmd.Env = repo, userEnv()
			out, _ := cmd.CombinedOutput()
			sees(t, string(out), cmd.ProcessState.ExitCode(), 1, "preserve existing hook")
			if data, _ := os.ReadFile(filepath.Join(repo, ".git", "hooks", "pre-commit")); string(data) != mine {
				t.Errorf("--install-hook replaced the existing hook:\n%s", data)
			}
		}},
		{"a clone whose hooks path is .githooks in a repo that ships .fitness-hooks", func(t *testing.T) {
			t.Parallel()
			repo, env := hpStale(t, ".githooks", ".githooks", ".fitness-hooks")
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
			sees(t, out, code, 0)
			hpWants(t, repo, ".fitness-hooks")
		}},
		{"a clone whose hooks path names a folder the repo does not ship", func(t *testing.T) {
			t.Parallel()
			repo, env := hpStale(t, "old-hooks", ".githooks")
			write(t, repo, hpShims("old-hooks"))
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
			sees(t, out, code, 0)
			hpWants(t, repo, ".githooks")
		}},
		{"a clone whose hooks path is outside the repo", func(t *testing.T) {
			t.Parallel()
			outside := t.TempDir()
			write(t, outside, hpShims("hooks"))
			repo, env := hpStale(t, outside+"/hooks", ".fitness-hooks")
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
			sees(t, out, code, 0, "core.hooksPath "+outside+"/hooks is outside the repo")
			hpWants(t, repo, outside+"/hooks")
		}},
		{"a clone whose hooks path still names .githooks", func(t *testing.T) {
			t.Parallel()
			repo, env := hpStale(t, ".githooks", ".githooks", ".fitness-hooks")
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
			sees(t, out, code, 1, "fitness hook: core.hooksPath was .githooks, now .fitness-hooks", `Commit message: "added some stuff"`)
			if strings.Count(out, "core.hooksPath") != 1 {
				t.Errorf("want one line naming the hooks path:\n%s", out)
			}
			if strings.Index(out, "now .fitness-hooks") > strings.Index(out, `Commit message: "added some stuff"`) {
				t.Errorf("the repair line comes after the hook's errors:\n%s", out)
			}
		}},
		{"a clone with a stale hooks path in a repo that ships .fitness-hooks", func(t *testing.T) {
			t.Parallel()
			repo, env := hpStale(t, ".githooks", ".githooks", ".fitness-hooks")
			out, code := fitness(t, repo, env, "init")
			sees(t, out, code, 0, "set core.hooksPath to .fitness-hooks")
			hpWants(t, repo, ".fitness-hooks")
		}},
		{"a clone with a stale hooks path, and a git that stalls reading it or the shipped folders", func(t *testing.T) {
			t.Parallel()
			for _, call := range []string{"core.hooksPath", "ls-tree"} {
				repo, env := hpStale(t, ".githooks", ".githooks", ".fitness-hooks")
				write(t, repo, map[string]string{".git/COMMIT_EDITMSG": "docs(readme): touch nothing\n"})
				path, pidFile := promptingTools(t, call)
				start := time.Now()
				out, code := fitness(t, repo, append(env, path), "hook", "commit-msg", ".git/COMMIT_EDITMSG")
				sees(t, out, code, 0)
				if took := time.Since(start); took > 30*time.Second {
					t.Errorf("a stalled git %s must end at its deadline, but the hook took %s", call, took)
				}
				hpWants(t, repo, ".githooks")
				processStopped(t, pidFile)
			}
		}},
	}.run(t)
}

func Test0005_2(t *testing.T) {
	cases{
		{"the hooks and a change that breaks a check", func(t *testing.T) {
			t.Parallel()
			repo, env := hooked(t, nil)
			write(t, repo, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
			user(t, repo, env, "add", "-A")
			out, code := user(t, repo, env, "commit", "-m", "docs(readme): add bold text")
			sees(t, out, code, 1, "README.md: disallowed **bold**")
		}},
		{"the hooks and a message that is not a semantic commit", func(t *testing.T) {
			t.Parallel()
			repo, env := hooked(t, nil)
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
			sees(t, out, code, 1, `Commit message: "added some stuff"`)
		}},
		{"the hooks and a message that is not a semantic commit and misspells its Plan trailer", func(t *testing.T) {
			t.Parallel()
			repo, env := hooked(t, nil)
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff\n\nPlan 63")
			sees(t, out, code, 1, `Commit message: "added some stuff"`, `plan trailer "Plan 63" must read exactly "Plan #<number>"`)
		}},
		{"the hooks and a failing executable pre-commit.local, or one whose git waits for a password", func(t *testing.T) {
			t.Parallel()
			repo, env := hooked(t, nil)
			write(t, repo, map[string]string{".githooks/pre-commit.local": "#!/bin/sh\necho local build broke\nexit 1\n"})
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
			sees(t, out, code, 1, "local build broke")
			localHookStops(t)
		}},
		{"the Go hook installed with --install-hook and a GitHub remote", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			git(t, repo, "remote", "add", "origin", "https://github.com/"+repoPR+".git")
			rp := standIn(t, map[string][]response{"gh": nil})
			cmd := exec.Command(installer(t), "--install-hook")
			cmd.Dir, cmd.Env = repo, userEnv()
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("--install-hook: %v\n%s", err, out)
			}
			write(t, repo, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
			user(t, repo, rp.env, "add", "-A")
			out, code := user(t, repo, rp.env, "commit", "-m", "docs(readme): add bold text")
			sees(t, out, code, 1, "README.md: disallowed **bold**")
			if len(rp.calls("gh")) != 0 {
				t.Fatalf("commit hook queried PR metadata: %q", rp.calls("gh"))
			}
		}},
		{"the hooks, external policy, and disabledChecks naming semantic-commit", func(t *testing.T) {
			t.Parallel()
			repo, env := hooked(t, map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["semantic-commit"]}` + "\n"})
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "added some stuff")
			sees(t, out, code, 0)
		}},
		{"the shared hooks and a GitHub branch", func(t *testing.T) {
			t.Parallel()
			repo, hookEnv := hooked(t, nil)
			git(t, repo, "remote", "add", "origin", "https://github.com/"+repoPR+".git")
			rp := standIn(t, map[string][]response{"gh": nil})
			out, code := user(t, repo, pushEnv(hookEnv, rp), "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
			sees(t, out, code, 0, "docs(readme): touch nothing")
			if strings.Contains(read(out), "local-branch passed") || strings.Contains(read(out), "local-branch failed") || len(rp.calls("gh")) != rp.count("gh", "auth token") {
				t.Fatalf("pre-commit ran metadata: %s\n%q", out, rp.calls("gh"))
			}
		}},
		{"the hooks and a branch with a commit of its own", func(t *testing.T) {
			t.Parallel()
			repo, env := hooked(t, nil)
			git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
			write(t, repo, noteGuide(t))
			git(t, repo, "add", "CHANGELOG.md")
			git(t, repo, "-c", "core.hooksPath=", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "docs(changelog): note the guide")
			write(t, repo, map[string]string{"README.md": readme + "\nMore words.\n"})
			git(t, repo, "add", "README.md")
			out, _ := user(t, repo, append(env, prEnv...), "commit", "-q", "-m", "docs(readme): add words")
			if !strings.Contains(out, "Running checks:") || strings.Contains(out, "Stage 1 of") {
				t.Errorf("the pre-commit hook must run the checks once, in no stages:\n%s", out)
			}
		}},
	}.run(t)
}

func Test0005_3(t *testing.T) {
	cases{
		{"a branch commit that links an issue", func(t *testing.T) {
			t.Parallel()
			repo, env := linked(t)
			out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): again\n\nPlan #218")
			sees(t, out, code, 1, `#218 is already linked by commit`, `("chore(app): start")`)
		}},
		{"a branch commit that links an issue: the commit goes through", func(t *testing.T) {
			t.Parallel()
			repo, env := linked(t)
			out, code := user(t, repo, env, "commit", "--amend", "--allow-empty", "-m", "chore(app): start over\n\nPlan #218")
			sees(t, out, code, 0, "chore(app): start over")
		}},
		{"a branch stacked on a pushed epic branch whose commit links an issue", func(t *testing.T) {
			t.Parallel()
			repo, env := linked(t)
			user(t, repo, env, "push", "-q", "origin", "HEAD:epic")
			user(t, repo, env, "switch", "-q", "-c", "child")
			out, code := user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "chore(app): child\n\nPlan #218")
			sees(t, out, code, 0)
			out, code = user(t, repo, env, "commit", "--allow-empty", "-m", "chore(app): again\n\nPlan #218")
			sees(t, out, code, 1, `#218 is already linked by commit`, `("chore(app): child")`)
		}},
		{"is behind a proxy that never answers, on a branch whose commit links an issue", func(t *testing.T) {
			t.Parallel()
			repo, env := linked(t)
			recordedNow(t, env)
			out, code := silentCommit(t, repo, env, "chore(app): again\n\nPlan #218")
			sees(t, out, code, 1, `#218 is already linked by commit`, offlineNote, "so fitness skipped the open PR read without connecting")
			out, code = silentCommit(t, repo, env, "chore(app): next\n\nPlan #219")
			sees(t, out, code, 0, offlineNote, "chore(app): next")
		}},
	}.run(t)
}

func Test0005_4(t *testing.T) {
	cases{
		{"the shared hooks, no fitness on the PATH, and a branch whose open PR closes an approved Plan", func(t *testing.T) {
			t.Parallel()
			out, code, rp := pushClosing(t, "Closes #218")
			sees(t, out, code, 0, "topic -> topic")
			pushAsked(t, rp, "218", "comments")
			out, code, _ = pushClosing(t, "Fixes #308")
			sees(t, out, code, 0, "topic -> topic")
			for plan := 309; plan <= 320; plan++ {
				out, code, _ = pushClosing(t, "Closes #"+strconv.Itoa(plan))
				sees(t, out, code, 0, "topic -> topic")
			}
			out, code, _ = pushFeature(t, listed, "Resolves #307", "feat(app): add a feature")
			sees(t, out, code, 0, "topic -> topic")
		}},
		{"a feature branch with no open PR yet", func(t *testing.T) {
			t.Parallel()
			out, code, rp := pushClosing(t, "")
			sees(t, out, code, 0, "pre-push: info: topic has no open PR yet, so its PR check gates the Plan.", "topic -> topic")
			if !rp.called("gh", "head=may-journal%3Atopic") || rp.called("gh", "issue view") {
				t.Errorf("want one PR lookup for topic and no Plan read; gh calls: %q", rp.calls("gh"))
			}
		}},
		{"pushed a branch", func(t *testing.T) {
			t.Parallel()
			repo, env, _ := pushRepo(t, gateOnly, "")
			user(t, repo, env, "push", "-q", "origin", "HEAD:topic")
			out, code := user(t, repo, env, "push", "origin", "--delete", "topic")
			sees(t, out, code, 0, "[deleted]", "topic")
		}},
		{"only chore or docs commits, even with feature commits merged in from main", func(t *testing.T) {
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
		}},
		{"feature commits on a branch whose PR closes one approved Plan twice, or a network that never answers", func(t *testing.T) {
			t.Parallel()
			repo, env, rp := pushRepo(t, gateOnly, "Closes #218\n\nFixes #218")
			user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): start")
			user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "feat(app): continue")
			user(t, repo, env, "commit", "-q", "--allow-empty", "-m", "fix(app): follow up")
			out, code := user(t, repo, env, "push", "origin", "topic")
			sees(t, out, code, 0, "topic -> topic")
			planReadOnce(t, rp, "218")
			silentPushRefused(t)
		}},
		{"external policy with plan-trailer and local-branch in disabledChecks", func(t *testing.T) {
			t.Parallel()
			off := `{"policy": "external", "disabledChecks": ["plan-trailer", "local-branch"]}` + "\n"
			repo, env, rp := pushRepo(t, off, "Refs #218")
			user(t, repo, env, "commit", "--allow-empty", "-m", "feat(app): add a feature")
			out, code := user(t, repo, env, "push", "origin", "topic")
			sees(t, out, code, 0, "topic -> topic")
			if rp.called("gh", "head=may-journal%3Atopic") {
				t.Errorf("the gate ran with plan-trailer off; gh calls: %q", rp.calls("gh"))
			}
		}},
		{"a branch whose PR closes an approved Plan of origin's owner in another repo, as `Closes May-Journal/.github#144` or its URL", func(t *testing.T) {
			t.Parallel()
			out, code, rp := pushClosing(t, "Closes May-Journal/.github#144")
			sees(t, out, code, 0, "topic -> topic")
			if n := rp.count("gh", "issue view 144 ", "-R May-Journal/.github"); n != 1 {
				t.Errorf("want one read of May-Journal/.github#144, got %d; gh calls: %q", n, rp.calls("gh"))
			}
			out, code, rp = pushClosing(t, "Fixes https://github.com/may-journal/.github/issues/144")
			sees(t, out, code, 0, "topic -> topic")
			pushAsked(t, rp, "144", "-R may-journal/.github")
		}},
		{"a branch whose PR closes no Plan despite a `Plan #NN` trailer, or a non-Plan issue, foreign or unapproved Plan", func(t *testing.T) {
			t.Parallel()
			out, code, _ := pushFeature(t, gateOnly, "Refs #218, and `Closes #218` in code.", "feat(app): add a feature\n\nPlan #218")
			sees(t, out, code, 1, "pre-push: https://github.com/"+repoPR+"/pull/1 closes no Plan.",
				"The PR body must close the approved Plan: 'Closes #NN', or 'Closes <owner>/<repo>#NN' for a Plan in another repo")
			out, code, rp := pushClosing(t, "Closes #196")
			sees(t, out, code, 1, "pre-push: no approved Plan Issue among (196)")
			pushAsked(t, rp, "196", "labels")
			out, code, rp = pushClosing(t, "Closes #151")
			sees(t, out, code, 1, "pre-push: no approved Plan Issue among (151)",
				`A Plan needs a comment saying approved, in any form or with a typo, from the repo's owner or an org member (planApprovers is unset).`)
			pushAsked(t, rp, "151", "comments")
			out, code, _ = pushClosing(t, "Closes #307")
			sees(t, out, code, 1, "pre-push: no approved Plan Issue among (307)")
			for plan := 321; plan <= 330; plan++ {
				out, code, _ = pushClosing(t, "Closes #"+strconv.Itoa(plan))
				sees(t, out, code, 1, "pre-push: no approved Plan Issue among ("+strconv.Itoa(plan)+")")
			}
			out, code, _ = pushFeature(t, listed, "Closes #218", "feat(app): add a feature")
			sees(t, out, code, 1, "pre-push: no approved Plan Issue among (218)",
				`A Plan needs a comment saying approved, in any form or with a typo, from one of planApprovers: Reviewer-Bot.`)
			out, code, rp = pushClosing(t, "Closes may-journal/.github#141")
			sees(t, out, code, 1, "no approved Plan Issue among (may-journal/.github#141)")
			pushAsked(t, rp, "141", "-R may-journal/.github")
			out, code, rp = pushClosing(t, "Closes other-org/.github#7")
			sees(t, out, code, 1, "no approved Plan Issue among (other-org/.github#7)", `Plan other-org/.github#7 is refused: only a Plan owned by "may-journal" counts here.`)
			if rp.called("gh", "issue view 7 ") {
				t.Errorf("pre-push read another owner's Plan; gh calls: %q", rp.calls("gh"))
			}
		}},
	}.run(t)
}

func Test0005_5(t *testing.T) {
	cases{
		{"file violations and a PR with a bad title", func(t *testing.T) {
			t.Parallel()
			repo := branchRepo(t, map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
			pr := branchAnswer(t, repo, "added one file", prPR(11))
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issuePR(t, 11, planBody("- [x] Write the tests."), "Plan"))})
			out, code := fitness(t, repo, rp.env)
			sees(t, out, code, 1, "README.md: disallowed **bold**", `"added one file"`, "pr-check #1")
			if !strings.Contains(row(out, "local-branch"), "failed") {
				t.Fatalf("full suite omitted the metadata result: %s", out)
			}
			branchReadOnce(t, rp)
		}},
		{"no open PR on the branch", func(t *testing.T) {
			t.Parallel()
			rp := standIn(t, map[string][]response{"gh": branchGitHub(response{Match: []string{"pulls?state=open"}})})
			out, code := fitness(t, branchRepo(t, nil), rp.env, "branch-check")
			sees(t, out, code, 0, "no open PR", "skipped")
		}},
		{"a detached HEAD", func(t *testing.T) {
			t.Parallel()
			repo := branchRepo(t, nil)
			git(t, repo, "checkout", "--detach", "HEAD")
			rp := standIn(t, map[string][]response{"gh": nil})
			out, code := fitness(t, repo, rp.env, "branch-check")
			sees(t, out, code, 0, "detached HEAD", "skipped")
			if len(rp.calls("gh")) != 0 {
				t.Fatalf("detached run queried GitHub: %q", rp.calls("gh"))
			}
		}},
		{"two open PRs matching the branch", func(t *testing.T) {
			t.Parallel()
			repo := branchRepo(t, nil)
			pr := branchAnswer(t, repo, "test(app): add one file", prPR(11))
			pr.Stdout += pr.Stdout
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr)})
			out, code := fitness(t, repo, rp.env, "branch-check")
			sees(t, out, code, 1, "multiple open PRs")
		}},
		{"credentials GitHub rejects", func(t *testing.T) {
			t.Parallel()
			pr := response{Match: []string{"pulls?state=open"}, Exit: 1, Stderr: "HTTP 401: authentication required"}
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr)})
			out, code := fitness(t, branchRepo(t, nil), rp.env, "branch-check")
			sees(t, out, code, 1, "HTTP 401", "retry fitness branch-check")
		}},
		{"a PR with no closing links", func(t *testing.T) {
			t.Parallel()
			repo := branchRepo(t, nil)
			body := strings.ReplaceAll(prPR(11), "Closes #11", "")
			rp := standIn(t, map[string][]response{"gh": branchGitHub(branchAnswer(t, repo, "test(app): add one file", body))})
			out, code := fitness(t, repo, rp.env, "branch-check")
			sees(t, out, code, 1, "no closing issue links", "closing")
		}},
		{"a fork whose stale main lacks an upstream Plan commit and a PR closing an ordinary issue", func(t *testing.T) {
			t.Parallel()
			repo, pr := forkBranch(t, "main")
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issuePR(t, 11, "An ordinary issue.", "bug"), issuePR(t, 12, planBody("- [x] Write the tests."), "Plan"))})
			out, code := fitness(t, repo, rp.env, "branch-check")
			sees(t, out, code, 0, "pr-check #1", "plan-check #12", "Issue: https://github.com/"+repoPR+"/issues/11")
			if rp.called("gh", "issues/999") {
				t.Fatalf("validated an upstream Plan outside the PR: %q", rp.calls("gh"))
			}
		}},
		{"a branch with a GitHub remote", func(t *testing.T) {
			t.Parallel()
			rp := standIn(t, map[string][]response{"gh": nil})
			out, code := fitness(t, branchRepo(t, nil), rp.env, "markdown-no-bold-italic")
			sees(t, out, code, 0, "passed")
			if len(rp.calls("gh")) != 0 {
				t.Fatalf("single check queried GitHub: %q", rp.calls("gh"))
			}
		}},
		{"an invalid PR and an explicit local-branch selection", func(t *testing.T) {
			t.Parallel()
			repo := branchRepo(t, nil)
			pr := branchAnswer(t, repo, "added one file", prPR(11))
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr, issuePR(t, 11, "An ordinary issue."))})
			out, code := fitness(t, repo, rp.env, "--policy=external", "--checks=markdown-no-bold-italic,local-branch")
			sees(t, out, code, 1, `"added one file"`, "1 of 2 checks passed")
			if !strings.Contains(row(out, "local-branch"), "failed") {
				t.Fatalf("missing metadata result row: %s", out)
			}
			branchReadOnce(t, rp)
		}},
		{"the installed local-branch binary, and its check package from the working tree in a module without the engine", func(t *testing.T) {
			t.Parallel()
			bin := filepath.Join(installOnly(t), "fitness-check-local-branch")
			rp := standIn(t, map[string][]response{"gh": nil})
			out, code := execute(t, func() *exec.Cmd {
				cmd := exec.Command(bin, "--describe")
				cmd.Env = append(userEnv(), rp.env...)
				return cmd
			})
			sees(t, out, code, 0, `"name":"local-branch"`, `"scope":"github"`)
			if len(rp.calls("gh")) != 0 {
				t.Fatalf("describe queried GitHub: %q", rp.calls("gh"))
			}
			// A build of the check alone, without the engine, cannot judge.
			mod := outsideModule(t, "example.com/alone", "package main\n\nimport check \"github.com/may-journal/fitness-runner/go/cmd/fitness-check-local-branch\"\n\nfunc main() { check.Main() }\n")
			goOutside(t, mod, "build the check alone", "build", "-o", "alone", ".")
			got, code := execute(t, func() *exec.Cmd { return exec.Command(filepath.Join(mod, "alone"), "--root", mod) })
			sees(t, got, code, 3, "runs only inside fitness")
		}},
		{"an Actions run in a GitHub checkout", func(t *testing.T) {
			t.Parallel()
			rp := standIn(t, map[string][]response{"gh": nil})
			out, code := fitness(t, branchRepo(t, nil), append(rp.env, "GITHUB_ACTIONS=true"))
			// Every installed check runs and passes, local-branch among them.
			sees(t, out, code, 0, allPassed())
			if len(rp.calls("gh")) != 0 {
				t.Fatalf("Actions ran local discovery: %q", rp.calls("gh"))
			}
		}},
		{"a repo without a GitHub origin", func(t *testing.T) {
			t.Parallel()
			rp := standIn(t, map[string][]response{"gh": nil})
			out, code := fitness(t, example(t, "happyRepo", nil), rp.env, "local-branch")
			sees(t, out, code, 0, "skipped")
			passedWithNoFiles(t, out, "local-branch")
			if len(rp.calls("gh")) != 0 {
				t.Fatalf("non-GitHub repo queried GitHub: %q", rp.calls("gh"))
			}
		}},
		{"a fork PR whose API base commit is missing locally", func(t *testing.T) {
			t.Parallel()
			repo, pr := forkBranch(t, "release")
			var data map[string]any
			mustDo(t, json.Unmarshal([]byte(pr.Stdout), &data))
			data["baseSHA"] = strings.Repeat("1", 40)
			raw, err := json.Marshal(data)
			mustDo(t, err)
			pr.Stdout = string(raw)
			rp := standIn(t, map[string][]response{"gh": branchGitHub(pr)})
			out, code := fitness(t, repo, rp.env, "local-branch")
			sees(t, out, code, 1, "fetch release from https://github.com/"+repoPR+".git", "retry fitness branch-check")
		}},
	}.run(t)
}

func Test0005_6(t *testing.T) {
	cases{
		{"an invalid PR on a branch that is not checked out", func(t *testing.T) {
			t.Parallel()
			repo, env, _ := metadataPushRepo(t, "added one file")
			checkoutMain(t, repo)
			pr := pushedAnswer(t, repo, "work", "added one file", prPR(11), 1)
			empty := response{Match: []string{"head=" + url.QueryEscape("may-journal:main")}}
			saved := pushedGitHub(t, planBody("- [x] Write the tests."), pr, empty)
			rp := standIn(t, map[string][]response{"gh": saved})
			out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work")
			sees(t, out, code, 1, `"added one file"`)
			branchReadOnce(t, rp)
			if rp.called("gh", empty.Match...) {
				t.Fatalf("checked the checkout instead of the pushed branch: %q", rp.calls("gh"))
			}
		}},
		{"a valid pushed branch and an invalid checked-out branch", func(t *testing.T) {
			t.Parallel()
			repo, env, _ := metadataPushRepo(t, "test(app): add one file")
			checkoutMain(t, repo)
			valid := pushedAnswer(t, repo, "work", "test(app): add one file", prPR(11), 1)
			invalid := pushedAnswer(t, repo, "main", "changed checkout file", prPR(11), 2)
			saved := pushedGitHub(t, planBody("- [x] Write the tests."), valid, invalid)
			rp := standIn(t, map[string][]response{"gh": saved})
			out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work")
			sees(t, out, code, 0, "work -> work")
			branchReadOnce(t, rp)
		}},
		{"two pushed branches and one invalid PR", func(t *testing.T) {
			t.Parallel()
			repo, env, _ := metadataPushRepo(t, "added one file")
			checkoutMain(t, repo)
			git(t, repo, "branch", "other")
			invalid := pushedAnswer(t, repo, "work", "added one file", prPR(11), 1)
			valid := pushedAnswer(t, repo, "other", "test(app): add another file", prPR(11), 2)
			saved := pushedGitHub(t, planBody("- [x] Write the tests."), invalid, valid)
			rp := standIn(t, map[string][]response{"gh": saved})
			out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work", "other")
			sees(t, out, code, 1, "pr-check #1", "pr-check #2", `"added one file"`)
			if strings.Count(out, "→ local-branch") != 1 || len(rp.calls("gh")) == 0 {
				t.Fatalf("want one meta-check invocation: %s", out)
			}
			for _, pr := range []response{invalid, valid} {
				if count := strings.Count(strings.Join(rp.calls("gh"), "\n"), pr.Match[0]); count != 1 {
					t.Errorf("want one discovery for %s, got %d", pr.Match[0], count)
				}
			}
		}},
		{"an existing remote branch", func(t *testing.T) {
			t.Parallel()
			repo, env, _ := metadataPushRepo(t, "test(app): add one file")
			out, code := user(t, repo, env, "push", "origin", "work")
			sees(t, out, code, 0, "work -> work")
			rp := standIn(t, map[string][]response{"gh": nil})
			out, code = user(t, repo, pushEnv(env, rp), "push", "origin", "--delete", "work")
			sees(t, out, code, 0, "deleted")
			if len(rp.calls("gh")) != 0 {
				t.Fatalf("deletion queried metadata: %q", rp.calls("gh"))
			}
		}},
		{"a renamed push whose PR closes a malformed approved Plan", func(t *testing.T) {
			t.Parallel()
			repo, env, _ := metadataPushRepo(t, "test(app): add one file")
			checkoutMain(t, repo)
			pr := pushedAnswer(t, repo, "renamed", "test(app): add one file", prPR(12)+"Closes #11\n", 1)
			saved := append(pushedGitHub(t, brokenPlan, pr), issuePR(t, 12, "An ordinary issue."))
			rp := standIn(t, map[string][]response{"gh": saved})
			out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work:renamed")
			sees(t, out, code, 1, "pr-check #1", "plan-check #11", "Background")
			branchReadOnce(t, rp)
		}},
		{"two pushed branches whose PRs close one approved Plan", func(t *testing.T) {
			t.Parallel()
			repo, env, _ := metadataPushRepo(t, "test(app): add one file")
			checkoutMain(t, repo)
			git(t, repo, "branch", "other")
			work := pushedAnswer(t, repo, "work", "test(app): add one file", prPR(11), 1)
			other := pushedAnswer(t, repo, "other", "test(app): add another file", prPR(11), 2)
			rp := standIn(t, map[string][]response{"gh": pushedGitHub(t, planBody("- [x] Write the tests."), work, other)})
			out, code := user(t, repo, pushEnv(env, rp), "push", "origin", "work", "other")
			sees(t, out, code, 0, "work -> work", "other -> other")
			planReadOnce(t, rp, "11")
		}},
		{"an approved feature Plan and an invalid linked PR, or a docs branch whose PR read outlasts the push's deadline", func(t *testing.T) {
			t.Parallel()
			out, code, rp := metadataPush(t, "added one file")
			sees(t, out, code, 1, `"added one file"`, "local-branch", "failed")
			branchReadOnce(t, rp)
			pushAsked(t, rp, "11", "comments")
			// The gate's Plan read takes 25 of the push's 30 seconds. The gate skips
			// the docs branch, so local-branch's 10-second PR discovery runs out.
			repo, env, rp := stalledPushRepo(t, "test(app): add one file", 25, 10)
			start := time.Now()
			out, code = user(t, repo, env, "push", "origin", "HEAD:work", "other")
			sees(t, out, code, 1, "local-branch", "GitHub reads timed out")
			if took := time.Since(start); took > 34*time.Second {
				t.Errorf("push took %s, past the 30-second deadline", took)
			}
			planReadOnce(t, rp, "11")
		}},
		{"an approved feature Plan and a valid linked PR", func(t *testing.T) {
			t.Parallel()
			out, code, rp := metadataPush(t, "test(app): add one file")
			sees(t, out, code, 0, "local-branch", "All 1 checks passed", "HEAD -> work")
			branchReadOnce(t, rp)
			pushAsked(t, rp, "11", "comments")
			planReadOnce(t, rp, "11")
		}},
		{"a repo pinning every check, none installed, and a PR with bold text", func(t *testing.T) {
			t.Parallel()
			for name, hook := range map[string]string{"synced": "", "unpinned": unpinnedPrePush} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					out, code := pushPrePush(t, hook)
					sees(t, out, code, 1, "pr-check #1", "**bold**")
					if strings.Contains(out, "no result from the check") {
						t.Errorf("the %s pre-push hook ran the PR body checks without installing them:\n%s", name, out)
					}
				})
			}
		}},
	}.run(t)
}

func Test0005_7(t *testing.T) {
	cases{
		{"a fitness repo and Claude wrote a file that fails a check", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			write(t, repo, map[string]string{"notes.md": misspelledNote})
			out, code := hookEvent(t, repo, editEvent(repo, "notes.md"))
			sees(t, out, code, 0, `"decision":"block"`, "fitness: cspell failed on notes.md", "[cspell] notes.md:1", "mispeled")
			write(t, repo, map[string]string{"pages/[id].md": misspelledNote, "pages/i.md": misspelledNote})
			out, code = hookEvent(t, repo, editEvent(repo, "pages/[id].md"))
			sees(t, out, code, 0, "[cspell] pages/[id].md:1")
			if strings.Contains(out, "pages/i.md") {
				t.Errorf("an edit to pages/[id].md judged pages/i.md too:\n%s", out)
			}
		}},
		{"a fitness repo with a failing Go test, and Claude edited a file", func(t *testing.T) {
			repo := example(t, "happyRepo", map[string]string{
				".fitnessrc.json": `{"policy": "external", "disabledChecks": ["requirements"]}` + "\n",
				"go.mod":          "module example.com/hooked\n\ngo 1.22\n",
				"hooked.go":       "// Package hooked holds one constant.\npackage hooked\n\n// Answer is the answer.\nconst Answer = 42\n",
				"hooked_test.go":  "package hooked\n\nimport \"testing\"\n\nfunc TestAnswer(t *testing.T) { t.Fatal(\"go-test ran\") }\n",
			})
			runner := filepath.Join(installOnly(t), "fitness")
			best := time.Hour
			for i := range 3 {
				file := fmt.Sprintf("note%d.md", i)
				write(t, repo, map[string]string{file: "A short note.\n"})
				out, code, took := timedHook(t, runner, repo, t.TempDir(), editEvent(repo, file))
				silentPass(t, out, code)
				best = min(best, took)
			}
			if best > 300*time.Millisecond {
				t.Errorf("the edit hook took %s at best; it must answer within 300ms", best)
			}
			out, code := hookEvent(t, repo, editEvent(repo, "hooked_test.go"))
			if strings.Contains(out, "go-test ran") {
				t.Errorf("an edit ran go-test, which is in the test stage:\n%s", out)
			}
			silentPass(t, out, code)
		}},
		{"a fitness repo with uncommitted files, any name, that fail a check", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			write(t, repo, map[string]string{"notes.md": misspelledNote, "café.md": misspelledNote})
			out, code := hookEvent(t, repo, stopEvent(false))
			sees(t, out, code, 0, `"decision":"block"`, "fitness: cspell failed on the changed files", "[cspell] notes.md:1", "[cspell] café.md:1")
			out, code = hookEvent(t, repo, stopEvent(true))
			silentPass(t, out, code)
		}},
		{"a repo with no fitness config and no fitness git hooks", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": ""})
			write(t, repo, map[string]string{"notes.md": misspelledNote})
			out, code := hookEvent(t, repo, editEvent(repo, "notes.md"))
			silentPass(t, out, code)
		}},
		{"the hooks in both the fitness skill and the settings", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			write(t, repo, map[string]string{"notes.md": misspelledNote})
			outs, codes := hookCopies(t, repo, editEvent(repo, "notes.md"))
			if outs[0] != "" {
				outs[0], outs[1], codes[0], codes[1] = outs[1], outs[0], codes[1], codes[0]
			}
			silentPass(t, outs[0], codes[0])
			sees(t, outs[1], codes[1], 0, `"decision":"block"`)
		}},
		{"a fitness repo, and a subagent edited an uncommitted file that fails a check", func(t *testing.T) {
			t.Parallel()
			repo, cache := example(t, "happyRepo", nil), t.TempDir()
			write(t, repo, map[string]string{"notes.md": misspelledNote, "theirs.md": misspelledNote})
			transcript := editTranscript(t, filepath.Join(repo, "notes.md"))
			out, code := hookEventIn(t, repo, cache, subagentStop("a1", 1, false, transcript))
			sees(t, out, code, 0, `"decision":"block"`, "fitness: cspell failed on the subagent's changed files", "[cspell] notes.md:1")
			if strings.Contains(out, "theirs.md") {
				t.Errorf("a subagent was blocked on a file it never edited:\n%s", out)
			}
			for try, active := range []bool{false, true} {
				out, code = hookEventIn(t, repo, cache, subagentStop("a1", try+2, active, transcript))
				silentPass(t, out, code)
			}
			out, code = hookEventIn(t, repo, cache, subagentStop("a2", 1, false, transcript))
			sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
			out, code = hookEventIn(t, repo, cache, subagentStop("a3", 1, false, filepath.Join(t.TempDir(), "none.jsonl")))
			silentPass(t, out, code)
		}},
		{"a fitness repo on a feature branch", func(t *testing.T) {
			repo := example(t, "happyRepo", map[string]string{"theirs.md": misspelledNote})
			onBranch(t, repo)
			out, code := hookEvent(t, repo, startEvent("startup"))
			silentPass(t, out, code)
			commit(t, repo, map[string]string{"committed.md": misspelledNote}, "docs(notes): add a note")
			write(t, repo, map[string]string{"notes.md": misspelledNote})
			runner := filepath.Join(installOnly(t), "fitness")
			best := time.Hour
			for _, source := range []string{"startup", "resume", "compact"} {
				out, code, took := timedHook(t, runner, repo, t.TempDir(), startEvent(source))
				sees(t, out, code, 0, `"hookEventName":"SessionStart"`, `"additionalContext"`, "fitness: cspell failed on the branch's changed files",
					"[cspell] notes.md:1", "[cspell] committed.md:1")
				if strings.Contains(out, "theirs.md") {
					t.Errorf("the start hook judged a file only the default branch changed:\n%s", out)
				}
				best = min(best, took)
			}
			if best > time.Second {
				t.Errorf("the start hook took %s at best; it must answer within one second", best)
			}
		}},
		{"a fitness repo, a hook log with a long and an unreadable line, and edits", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			cache := t.TempDir()
			long := fmt.Sprintf(`{"session":"s0","event":"Stop","error":%q}`, strings.Repeat("stderr ", 30000))
			mustDo(t, os.WriteFile(filepath.Join(cache, "claude-hooks.jsonl"), []byte(long+"\nnot json\n"), 0o600))
			write(t, repo, map[string]string{"notes.md": misspelledNote, "clean.md": "A short note.\n"})
			for _, event := range []map[string]any{editEvent(repo, "notes.md"), editEvent(repo, "clean.md"), {"hook_event_name": "SessionEnd"}} {
				event["session_id"] = "s1"
				hookEventIn(t, repo, cache, event)
			}
			lines := strings.Split(strings.TrimSpace(fiFile(t, filepath.Join(cache, "claude-hooks.jsonl"))), "\n")
			if len(lines) != 5 || !loggedEditsAndTotal(lines[2:]) {
				t.Errorf("want two edit lines and a SessionEnd total of 2 runs, 1 block, 1 failed check, 1 finding line, got:\n%s", strings.Join(lines[min(2, len(lines)):], "\n"))
			}
			if _, err := os.Stat(filepath.Join(repo, "claude-hooks.jsonl")); err == nil {
				t.Error("the hook log must stay out of the repo")
			}
		}},
		{"a fitness repo with red uncommitted files, on the default branch or detached", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", nil)
			git(t, repo, "checkout", "-q", "-B", "main")
			git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
			write(t, repo, map[string]string{"notes.md": misspelledNote})
			out, code := hookEvent(t, repo, startEvent("startup"))
			silentPass(t, out, code)
			git(t, repo, "checkout", "-q", "--detach")
			out, code = hookEvent(t, repo, startEvent("resume"))
			silentPass(t, out, code)
		}},
		{"a fitness repo with a failing Go test, and a subagent edited a Go file", func(t *testing.T) {
			t.Parallel()
			installer(t)
			file := filepath.Join(t.TempDir(), "settings.json")
			mustDo(t, os.WriteFile(file, []byte("{}"), 0o600))
			hooks := claudeSettings(t, file)["hooks"].(map[string]any)
			if timeout := hookTimeout(hooks, "SubagentStop"); timeout != 300.0 {
				t.Errorf("SubagentStop's timeout must be 300 seconds, under Claude Code's ten-minute default, got %v", timeout)
			}
			if timeout := hookTimeout(hooks, "Stop"); timeout != 540.0 {
				t.Errorf("Stop's timeout must be 540 seconds, the 60s fast total and 450s test total with a margin, got %v", timeout)
			}
			repo := example(t, "happyRepo", failingGoTest)
			write(t, repo, map[string]string{"notes.md": misspelledNote, "hooked_test.go": strings.ReplaceAll(failingGoTest["hooked_test.go"], "ran", "ran again")})
			transcript := editTranscript(t, filepath.Join(repo, "hooked_test.go"), filepath.Join(repo, "notes.md"))
			out, code := hookEvent(t, repo, subagentStop("a1", 1, false, transcript))
			sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
			if strings.Contains(out, "go-test") {
				t.Errorf("a subagent stop ran go-test, which is in the test stage:\n%s", out)
			}
		}},
		{"a fitness repo with uncommitted files that fail a check", func(t *testing.T) {
			t.Parallel()
			repo, cache := example(t, "happyRepo", nil), t.TempDir()
			transcript := filepath.Join(t.TempDir(), "session.jsonl")
			write(t, repo, map[string]string{"notes.md": misspelledNote})
			addUserLine(t, transcript, "p1", "Fix the notes.")
			out, code := hookEventIn(t, repo, cache, claudeStop(1, transcript))
			sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
			addUserLine(t, transcript, "p1", "Stop hook feedback:\n"+out)
			out, code = hookEventIn(t, repo, cache, claudeStop(2, transcript))
			silentPass(t, out, code)
			addUserLine(t, transcript, "p2", "Now the readme.")
			out, code = hookEventIn(t, repo, cache, claudeStop(3, transcript))
			sees(t, out, code, 0, `"decision":"block"`, "[cspell] notes.md:1")
		}},
		{"a fitness repo on a feature branch whose edit checks outlast the start hook's 15 second deadline", func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", slowChecks())
			onBranch(t, repo)
			write(t, repo, map[string]string{"notes.md": misspelledNote})
			out, code, took := timedHook(t, filepath.Join(installOnly(t), "fitness"), repo, t.TempDir(), startEvent("startup"))
			sees(t, out, code, 0, `"additionalContext"`, "[cspell] notes.md:1", "fitness: skipped slow-", "past the hook's deadline")
			if took > 20*time.Second {
				t.Errorf("the start hook took %s; past its deadline it must answer within 20 seconds", took)
			}
		}},
	}.runAlone(t)
}
