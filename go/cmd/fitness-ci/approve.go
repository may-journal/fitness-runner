package main

import (
	"encoding/json"
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

// checkKind fails unless KIND names an approval Release approve gives.
func checkKind() error {
	switch kind := env("KIND"); kind {
	case "release", "dependency":
		return nil
	default:
		return actions.Fail(fmt.Sprintf("Release approve got kind '%s'; pass release or dependency.", kind))
	}
}

func checkApproverToken(step actions.Step, _ []string) error {
	if err := checkKind(); err != nil {
		return err
	}
	token := env("TOKEN")
	if token == "" {
		actions.Notice("APPROVER_TOKEN is not available to this repo, so a person approves the PR.")
		return nil
	}
	login, problem, err := tokenLogin(token)
	if err != nil {
		return actions.Fail(tokenProblem(problem))
	}
	return writeAll(step, "ready", "true", "login", login)
}

// tokenLogin reads token's login and checks it can see the repo, returning
// the error text of a failed call.
func tokenLogin(token string) (login, problem string, err error) {
	gh := actions.Call{Env: []string{"GH_TOKEN=" + token}}
	login, problem, err = gh.Quiet("gh", "api", "user", "--jq", ".login")
	if err == nil {
		_, problem, err = gh.Quiet("gh", "api", "repos/"+env("GITHUB_REPOSITORY"), "--silent")
	}
	return login, problem, err
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

// prView is the part of `gh pr view --json` the approval checks read.
type prView struct {
	Title  string `json:"title"`
	State  string `json:"state"`
	Draft  bool   `json:"isDraft"`
	Head   string `json:"headRefOid"`
	Base   string `json:"baseRefOid"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

// readyToApprove reports whether the PR is open and not a draft, noting
// when it is not.
func (v prView) readyToApprove(pr string) bool {
	if v.State == "OPEN" && !v.Draft {
		return true
	}
	actions.Notice("PR #" + pr + " is not open and ready, so it gets no approval.")
	return false
}

// changedFile is one file a PR changes, with its old name when renamed.
type changedFile struct {
	Filename string `json:"filename"`
	Previous string `json:"previous_filename"`
}

// changedFiles lists every file the PR changes, renamed ones under both names.
func changedFiles(pr string) ([]string, error) {
	files, err := actions.GhPages[changedFile]("api", "repos/"+env("GITHUB_REPOSITORY")+"/pulls/"+pr+"/files", "--paginate")
	var names []string
	for _, f := range files {
		names = append(names, f.Filename)
		names = append(names, nonEmpty(f.Previous)...)
	}
	return names, err
}

// nonEmpty is s as a one-item list, or none when s is empty.
func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

// onlyFiles reports whether pr changes some files and only allowed ones,
// noting the others when it does not. An error reads as false.
func onlyFiles(pr, what string, allowed ...string) (bool, error) {
	files, err := changedFiles(pr)
	if err != nil {
		return false, err
	}
	others := slices.DeleteFunc(slices.Clone(files), func(f string) bool { return slices.Contains(allowed, f) })
	if len(files) == 0 || len(others) > 0 {
		actions.Notice(fmt.Sprintf("PR #%s changes more than %s, so a person approves it: %s", pr, what, strings.Join(others, ", ")))
		return false, nil
	}
	return true, nil
}

func checkReleasePR(step actions.Step, _ []string) error {
	pr := env("PR")
	var view prView
	if err := actions.GhJSON(&view, "pr", "view", pr, "--json", "author,state,isDraft,headRefOid"); err != nil {
		return err
	}
	if author := env("AUTHOR"); view.Author.Login != author {
		actions.Notice(fmt.Sprintf("PR #%s is by %s, not %s, so a person approves it.", pr, view.Author.Login, author))
		return nil
	}
	if !view.readyToApprove(pr) {
		return nil
	}
	if ok, err := onlyFiles(pr, "the release files", "CHANGELOG.md", "version.txt", ".release-please-manifest.json"); !ok {
		return err
	}
	return step.Output("sha", view.Head)
}

// dependencyPR reads pr and reports whether it is an open, ready build(deps)
// PR, noting why when it is not.
func dependencyPR(pr string) (prView, bool, error) {
	var view prView
	if err := actions.GhJSON(&view, "pr", "view", pr, "--json", "title,state,isDraft,headRefOid,baseRefOid"); err != nil {
		return view, false, err
	}
	if !strings.HasPrefix(view.Title, "build(deps)") {
		actions.Notice("PR #" + pr + " is not titled build(deps), so a person approves it.")
		return view, false, nil
	}
	return view, view.readyToApprove(pr), nil
}

func checkDependencyBump(step actions.Step, _ []string) error {
	pr := env("PR")
	view, ok, err := dependencyPR(pr)
	if !ok {
		return err
	}
	if ok, err := onlyFiles(pr, "package.json and package-lock.json", "package.json", "package-lock.json"); !ok {
		return err
	}
	base, head, err := readSides("repos/"+env("GITHUB_REPOSITORY"), view)
	if err != nil {
		return err
	}
	if problems := bumpProblems(base, head); len(problems) > 0 {
		actions.Notice(fmt.Sprintf("PR #%s is not a patch or minor dev-dependency bump, so a person approves it: %s", pr, strings.Join(problems, "; ")))
		return nil
	}
	return step.Output("sha", view.Head)
}

// readSides reads the npm files at the merge base, as the file list compares,
// and at the PR's head.
func readSides(repo string, view prView) (base, head npmSide, err error) {
	sha, err := actions.Gh("api", repo+"/compare/"+view.Base+"..."+view.Head, "--jq", ".merge_base_commit.sha")
	if err != nil {
		return base, head, err
	}
	if base, err = readSide(repo, sha); err != nil {
		return base, head, err
	}
	head, err = readSide(repo, view.Head)
	return base, head, err
}

// readSide reads package.json and package-lock.json at commit sha.
func readSide(repo, sha string) (npmSide, error) {
	var files [2][]byte
	for i, file := range []string{"package.json", "package-lock.json"} {
		var err error
		if files[i], err = contentAt(repo, file, sha); err != nil {
			return npmSide{}, err
		}
	}
	return readNPMSide(files[0], files[1])
}

// contentAt reads file at commit sha. A file missing there reads as {}; any
// other error fails the step.
func contentAt(repo, file, sha string) ([]byte, error) {
	out, problem, err := actions.Call{}.Quiet("gh", "api", "-H", "Accept: application/vnd.github.raw+json", repo+"/contents/"+file+"?ref="+sha)
	switch {
	case err == nil:
		return []byte(out), nil
	case strings.Contains(problem, "HTTP 404"):
		return []byte("{}"), nil
	}
	return nil, actions.Fail(fmt.Sprintf("Could not read %s at %s: %s", file, sha, problem))
}

func waitForGreenChecks(step actions.Step, _ []string) error {
	pr := env("PR")
	tries, wait, err := triesAndWait()
	if err != nil {
		return err
	}
	verdict, err := waitVerdict(pr, tries, wait)
	if err != nil {
		return err
	}
	return approveIfGreen(step, pr, verdict, tries)
}

// triesAndWait reads how often to read the checks and how long to wait
// between reads.
func triesAndWait() (tries int, wait time.Duration, err error) {
	if tries, err = strconv.Atoi(env("TRIES")); err != nil {
		return 0, 0, fmt.Errorf("TRIES: %w", err)
	}
	seconds, err := strconv.Atoi(env("WAIT_SECONDS"))
	return tries, time.Duration(seconds) * time.Second, err
}

// waitVerdict reads the checks up to tries times until they are no longer
// pending, and returns green, red or pending.
func waitVerdict(pr string, tries int, wait time.Duration) (string, error) {
	verdict := "pending"
	for try := 1; try <= tries && verdict == "pending"; try++ {
		if try > 1 {
			time.Sleep(wait)
		}
		var err error
		if verdict, err = readVerdict(pr); err != nil {
			return "", err
		}
	}
	return verdict, nil
}

// prCheck is one check as `gh pr checks --json` lists it.
type prCheck struct {
	Bucket string `json:"bucket"`
	Link   string `json:"link"`
}

// readVerdict reads the PR's checks once. gh exits 1 on a failed check and
// 8 on a pending one, and prints the list either way.
func readVerdict(pr string) (string, error) {
	out, problem, _ := actions.Call{}.Quiet("gh", "pr", "checks", pr, "--json", "bucket,link")
	var checks []prCheck
	if json.Unmarshal([]byte(out), &checks) == nil && strings.HasPrefix(strings.TrimSpace(out), "[") {
		return verdictOf(buckets(checks)), nil
	}
	if strings.Contains(problem, "no checks reported") {
		return "pending", nil
	}
	return "", actions.Fail("Could not read the checks of PR #" + pr + ": " + problem)
}

// buckets lists each check's bucket but this run's own, known by its link.
func buckets(checks []prCheck) []string {
	own := "/actions/runs/" + env("GITHUB_RUN_ID") + "/"
	var kept []string
	for _, c := range checks {
		if !strings.Contains(c.Link, own) {
			kept = append(kept, c.Bucket)
		}
	}
	return kept
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

// approveIfGreen writes green when the checks passed at the commit checked.
func approveIfGreen(step actions.Step, pr, verdict string, tries int) error {
	switch verdict {
	case "red":
		actions.Notice("PR #" + pr + " has a failed or cancelled check, so it gets no approval.")
		return nil
	case "pending":
		actions.Warning(fmt.Sprintf("PR #%s still had checks running after %d tries, so a person approves it.", pr, tries))
		return nil
	}
	head, err := actions.Gh("pr", "view", pr, "--json", "headRefOid", "--jq", ".headRefOid")
	if err != nil {
		return err
	}
	if head != env("SHA") {
		actions.Notice(fmt.Sprintf("PR #%s moved to %s while its checks ran, so this run approves nothing.", pr, head))
		return nil
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
	pr, sha, bot := env("PR"), env("SHA"), env("BOT")
	reviews := "repos/" + env("GITHUB_REPOSITORY") + "/pulls/" + pr + "/reviews"
	listed, err := actions.GhPages[review]("api", reviews, "--paginate")
	if err != nil {
		return err
	}
	if slices.ContainsFunc(listed, func(r review) bool { return r.User.Login == bot && r.CommitID == sha && r.State == "APPROVED" }) {
		fmt.Printf("PR #%s is already approved at %s.\n", pr, sha)
		return nil
	}
	if _, err := actions.Gh("api", "-X", "POST", reviews, "-f", "event=APPROVE", "-f", "commit_id="+sha,
		"-f", "body=Approved: this PR changes only what Release approve allows for its kind, and its checks are green."); err != nil {
		return err
	}
	fmt.Printf("Approved PR #%s at %s.\n", pr, sha)
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
