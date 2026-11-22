package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

func init() {
	register("check-approver-token", "Check the approver token in TOKEN for the KIND of approval, writing ready and login when it is usable.", []string{"ready", "login"}, checkApproverToken)
	register("check-release-pr", "Pass a release PR the automation App opened that changes only release files, writing the commit it checked as sha.", []string{"sha"}, checkReleasePR)
	register("check-dependency-bump", "Pass a build(deps) PR that moves only dev packages by a patch or minor bump, writing the commit it checked as sha.", []string{"sha"}, checkDependencyBump)
	register("wait-for-green-checks", "Wait for every check but this run's own to pass, writing green when the PR is ready to approve.", []string{"green"}, waitForGreenChecks)
	register("approve-pr", "Approve the PR at the checked commit unless the approver already did.", nil, approvePR)
}

func checkApproverToken(step actions.Step, _ []string) error {
	if kind := env("KIND"); kind != "release" && kind != "dependency" {
		return actions.Fail(fmt.Sprintf("Release approve got kind '%s'; pass release or dependency.", kind))
	}
	if env("TOKEN") == "" {
		return actions.Skip("APPROVER_TOKEN is not available to this repo, so a person approves the PR.")
	}
	login, err := tokenLogin()
	if err != nil {
		return err
	}
	return writeAll(step, "ready", "true", "login", login)
}

// tokenLogin is TOKEN's login, once the token can see the repository.
func tokenLogin() (string, error) {
	gh := actions.Call{Env: []string{"GH_TOKEN=" + env("TOKEN")}}
	login, problem, err := gh.Quiet("gh", "api", "user", "--jq", ".login")
	if err == nil {
		_, problem, err = gh.Quiet("gh", "api", actions.RepoPath(), "--silent")
	}
	if err != nil {
		return "", actions.Fail(tokenProblem(problem))
	}
	return login, nil
}

// tokenProblem says what to do about the approver token gh refused.
func tokenProblem(problem string) string {
	switch {
	case strings.Contains(problem, "HTTP 401"):
		return "GitHub rejects APPROVER_TOKEN; replace the org secret with a valid token."
	case strings.Contains(problem, "HTTP 404"):
		return "APPROVER_TOKEN cannot see " + env("GITHUB_REPOSITORY") + "; give the token access to it."
	}
	return "Could not check APPROVER_TOKEN: " + problem
}

func checkReleasePR(step actions.Step, _ []string) error {
	pr := stepPR()
	v, err := pr.view("author,state,isDraft,headRefOid")
	if err != nil {
		return err
	}
	byApp := func() (string, error) {
		if author := env("AUTHOR"); v.Author.Login != author {
			return fmt.Sprintf("is by %s, not %s, so a person approves it.", v.Author.Login, author), nil
		}
		return "", nil
	}
	return passThen(step, v.Head)(pr.passes(byApp, openAndReady(v), pr.onlyChanges("the release files", "CHANGELOG.md", "version.txt", ".release-please-manifest.json")))
}

func checkDependencyBump(step actions.Step, _ []string) error {
	pr := stepPR()
	v, err := pr.view("title,state,isDraft,headRefOid,baseRefOid")
	if err != nil {
		return err
	}
	titled := func() (string, error) {
		if strings.HasPrefix(v.Title, "build(deps)") {
			return "", nil
		}
		return "is not titled build(deps), so a person approves it.", nil
	}
	return passThen(step, v.Head)(pr.passes(titled, openAndReady(v), pr.onlyChanges("package.json and package-lock.json", "package.json", "package-lock.json"), devBump(v)))
}

// passThen writes the checked commit sha once every gate passed.
func passThen(step actions.Step, sha string) func(bool, error) error {
	return func(passed bool, err error) error {
		if !passed {
			return err
		}
		return step.Output("sha", sha)
	}
}

// devBump refuses a PR whose npm files change more than a dev patch or minor
// bump. It compares the head with the merge base, as the file list does.
func devBump(v prView) gate {
	return func() (string, error) {
		base, err := actions.Gh("api", actions.RepoPath()+"/compare/"+v.Base+"..."+v.Head, "--jq", ".merge_base_commit.sha")
		if err != nil {
			return "", err
		}
		from, err := readSide(base)
		if err != nil {
			return "", err
		}
		to, err := readSide(v.Head)
		if err != nil {
			return "", err
		}
		problems := bumpProblems(from, to)
		if len(problems) == 0 {
			return "", nil
		}
		return "is not a patch or minor dev-dependency bump, so a person approves it: " + strings.Join(problems, "; "), nil
	}
}

// readSide reads package.json and package-lock.json at commit sha. A file
// missing there reads as {}; any other error fails the step.
func readSide(sha string) (npmSide, error) {
	var files [2][]byte
	for i, file := range []string{"package.json", "package-lock.json"} {
		out, problem, err := actions.Call{}.Quiet("gh", "api", "-H", "Accept: application/vnd.github.raw+json", actions.RepoPath()+"/contents/"+file+"?ref="+sha)
		if err != nil && !strings.Contains(problem, "HTTP 404") {
			return npmSide{}, actions.Fail(fmt.Sprintf("Could not read %s at %s: %s", file, sha, problem))
		}
		// gh prints GitHub's error body on a 404, so a missing file reads as {}.
		files[i] = []byte(cmpOr(onlyIf(err == nil, out), "{}"))
	}
	return readNPMSide(files[0], files[1])
}

// onlyIf is s when ok holds, else "".
func onlyIf(ok bool, s string) string {
	if ok {
		return s
	}
	return ""
}

// cmpOr is s, or fallback when s is empty.
func cmpOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func waitForGreenChecks(step actions.Step, _ []string) error {
	pr := stepPR()
	tries, _ := strconv.Atoi(env("TRIES"))
	verdict, err := waitVerdict(pr, tries)
	switch {
	case err != nil:
		return err
	case verdict == "red":
		return actions.Skip("PR #" + pr.number + " has a failed or cancelled check, so it gets no approval.")
	case verdict == "pending":
		actions.Warning(fmt.Sprintf("PR #%s still had checks running after %d tries, so a person approves it.", pr.number, tries))
		return nil
	}
	return stillAt(step, pr)
}

// waitVerdict reads the checks up to tries times, WAIT_SECONDS apart, until
// they are no longer pending.
func waitVerdict(pr pullRequest, tries int) (verdict string, err error) {
	wait, _ := strconv.Atoi(env("WAIT_SECONDS"))
	verdict = "pending"
	for try := 1; try <= tries && verdict == "pending" && err == nil; try++ {
		time.Sleep(time.Duration(min(try-1, 1)*wait) * time.Second)
		verdict, err = checksVerdict(pr)
	}
	return verdict, err
}

// checksVerdict reads the PR's checks once, but this run's own, as red,
// pending or green. gh exits 1 on a failed check and 8 on a pending one,
// and prints the list either way.
func checksVerdict(pr pullRequest) (string, error) {
	out, problem, _ := actions.Call{}.Quiet("gh", "pr", "checks", pr.number, "--json", "bucket,link")
	checks, err := actions.Pages[struct{ Bucket, Link string }](out)
	if err != nil || !strings.HasPrefix(strings.TrimSpace(out), "[") {
		if strings.Contains(problem, "no checks reported") {
			return "pending", nil
		}
		return "", actions.Fail("Could not read the checks of PR #" + pr.number + ": " + problem)
	}
	own := "/actions/runs/" + env("GITHUB_RUN_ID") + "/"
	checks = slices.DeleteFunc(checks, func(c struct{ Bucket, Link string }) bool { return strings.Contains(c.Link, own) })
	buckets := make([]string, len(checks))
	for i, c := range checks {
		buckets[i] = c.Bucket
	}
	return verdictOf(buckets), nil
}

// verdictOf is red for any failed or cancelled check, pending while any runs
// or none exists, and green otherwise.
func verdictOf(buckets []string) string {
	switch {
	case slices.Contains(buckets, "fail") || slices.Contains(buckets, "cancel"):
		return "red"
	case len(buckets) == 0 || slices.Contains(buckets, "pending"):
		return "pending"
	}
	return "green"
}

// stillAt writes green unless the PR moved past the commit checked.
func stillAt(step actions.Step, pr pullRequest) error {
	head, err := pr.field("headRefOid")
	switch {
	case err != nil:
		return err
	case head != env("SHA"):
		return actions.Skip(fmt.Sprintf("PR #%s moved to %s while its checks ran, so this run approves nothing.", pr.number, head))
	}
	return step.Output("green", "true")
}

// review is one review as GitHub lists it.
type review struct {
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CommitID string `json:"commit_id"`
	State    string `json:"state"`
}

func approvePR(actions.Step, []string) error {
	pr, sha := stepPR(), env("SHA")
	listed, err := actions.GhPages[review]("api", pr.path()+"/reviews", "--paginate")
	if err != nil {
		return err
	}
	if slices.ContainsFunc(listed, func(r review) bool { return r.User.Login == env("BOT") && r.CommitID == sha && r.State == "APPROVED" }) {
		fmt.Printf("PR #%s is already approved at %s.\n", pr.number, sha)
		return nil
	}
	if _, err := actions.Gh("api", "-X", "POST", pr.path()+"/reviews", "-f", "event=APPROVE", "-f", "commit_id="+sha,
		"-f", "body=Approved: this PR changes only what Release approve allows for its kind, and its checks are green."); err != nil {
		return err
	}
	fmt.Printf("Approved PR #%s at %s.\n", pr.number, sha)
	return nil
}

// writeAll writes each key and value pair in pairs to the step output.
func writeAll(step actions.Step, pairs ...string) error {
	for i := 0; i+1 < len(pairs); i += 2 {
		if err := step.Output(pairs[i], pairs[i+1]); err != nil {
			return err
		}
	}
	return nil
}
