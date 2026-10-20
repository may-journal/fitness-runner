package requirements

import "testing"

func Test0101_1(t *testing.T) {
	t.Parallel()
	const quote = `plan trailer "Plan 63" ` + planTrailerFix
	taught(t, "plan-trailer", quote)
	out, code := planTrailer(t, "feat(app): add a feature\n\nPlan 63\nplan #63\nPlan: 63\n")
	sees(t, out, code, 1, quote)
	out, code = planTrailer(t, "feat(app): add a feature\n\nPlan #63\n")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0101_2(t *testing.T) {
	t.Parallel()
	quote := commitAttributionMissing("AI-Models")
	taught(t, "commit-attribution", quote)
	out, code := commitAttribution(t, []string{commitAttributionTrailers}, "--message=feat(api): add paging\n\nJust a body.\n")
	sees(t, out, code, 1, quote, commitAttributionMissing("AI-Tools"))
	out, code = commitAttribution(t, []string{commitAttributionTrailers}, "--message="+commitAttributionTrailers)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0101_3(t *testing.T) {
	t.Parallel()
	quote := `Commit message: "feat: add endpoint" — use type(scope): description ` + semanticCommitTypes
	taught(t, "semantic-commit", quote)
	out, code := semanticCommit(t, "--message", "feat: add endpoint")
	sees(t, out, code, 1, quote)
	out, code = semanticCommit(t, "--message", "feat(api): add endpoint")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0101_4(t *testing.T) {
	t.Parallel()
	quote := iloRepo + "#12 is already linked by commit"
	taught(t, "issue-link-once", quote)
	repo := iloOnGitHub(t, "topic")
	iloCommit(t, repo, "", "feat(app): start\n\nPlan #12")
	out, code := iloCheck(t, repo, "fix(app): more\n\nPlan #12")
	sees(t, out, code, 1, quote)
	out, code = iloCheck(t, repo, "fix(app): more")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0101_5(t *testing.T) {
	t.Parallel()
	taught(t, "local-branch", "no closing issue links")
	repo := branchRepo(t, nil)
	body := "> One clear summary of the change.\n\n## Background\n\nThis change is a test fixture.\n\n## Changelog\n\n" + link0014("CHANGELOG.md#"+stamp0014) + "\n"
	rp := standIn(t, map[string][]response{"gh": branchGitHub(branchAnswer(t, repo, "test(app): add one file", body))})
	out, code := fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 1, "no closing issue links")
	pr := branchAnswer(t, repo, "test(app): add one file", pr0014(11))
	rp = standIn(t, map[string][]response{"gh": branchGitHub(pr, issue0014(t, 11, "An ordinary issue."))})
	out, code = fitness(t, repo, rp.env, "branch-check")
	sees(t, out, code, 0, "pr-check #1")
}

func Test0101_6(t *testing.T) {
	t.Parallel()
	const quote = ".github/workflows/comment-check-reusable.yml does not exist at v1"
	taught(t, "workflow-ref-exists", quote)
	out, code, _ := refCheck(t, refWorkflow(
		"may-journal/fitness-runner/.github/workflows/comment-check-reusable.yml@v1",
	), []response{
		found("repos/may-journal/fitness-runner/git/ref/tags/v1", `{"ref":"refs/tags/v1"}`),
		notFound("repos/may-journal/fitness-runner/contents/.github/workflows/comment-check-reusable.yml?ref=v1"),
		found("repos/may-journal/fitness-runner", `{"full_name":"may-journal/fitness-runner"}`),
	})
	sees(t, out, code, 1, quote)
	out, code, _ = refCheck(t, refWorkflow(
		"may-journal/fitness-runner/.github/workflows/ci-reusable.yml@v1",
	), []response{
		found("repos/may-journal/fitness-runner/git/ref/tags/v1", `{"ref":"refs/tags/v1"}`),
		found("repos/may-journal/fitness-runner/contents/.github/workflows/ci-reusable.yml?ref=v1", `{"name":"ci-reusable.yml"}`),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0101_7(t *testing.T) {
	t.Parallel()
	const quote = "decisions logged in the repo"
	taught(t, "read-repo-first", quote)
	out, code := readRepoFirst(t)
	sees(t, out, code, 0, quote, "All 1 checks passed")
}
