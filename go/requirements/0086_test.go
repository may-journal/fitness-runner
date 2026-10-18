package requirements

import (
	"strings"
	"testing"
)

// refSha is a full commit sha a workflow pins an action to.
const refSha = "6b02ebef8736d2508d47e085806b5b3cd19efa6c"

// refWorkflow is a workflow file whose jobs call each `uses:` value, one
// job per value, so value i sits on line 4+2i.
func refWorkflow(uses ...string) string {
	var b strings.Builder
	b.WriteString("name: CI\non: push\njobs:\n")
	for i, u := range uses {
		b.WriteString("  job" + string(rune('a'+i)) + ":\n    uses: " + u + "\n")
	}
	return b.String()
}

// found is a saved GitHub answer for a path that exists.
func found(match, body string) response {
	return response{Match: []string{"api " + match}, Stdout: body}
}

// notFound is a saved GitHub answer for a path that does not exist.
func notFound(match string) response {
	return response{Match: []string{"api " + match}, Stderr: "gh: Not Found (HTTP 404)\n", Exit: 1}
}

// refCheck runs workflow-ref-exists on happyRepo with .github/workflows/ci.yml
// set to workflow, GitHub answering from saved responses.
func refCheck(t *testing.T, workflow string, saved []response) (string, int, replays) {
	t.Helper()
	rp := standIn(t, map[string][]response{"gh": saved})
	repo := example(t, "happyRepo", map[string]string{".github/workflows/ci.yml": workflow})
	out, code := fitness(t, repo, rp.env, "workflow-ref-exists")
	return out, code, rp
}

func Test0086_1(t *testing.T) {
	t.Parallel()
	out, code, _ := refCheck(t, refWorkflow(
		"may-journal/fitness-runner/.github/workflows/ci-reusable.yml@v1",
		"may-journal/fitness-runner@"+refSha,
	), []response{
		found("repos/may-journal/fitness-runner/git/ref/tags/v1", `{"ref":"refs/tags/v1"}`),
		found("repos/may-journal/fitness-runner/contents/.github/workflows/ci-reusable.yml?ref=v1", `{"name":"ci-reusable.yml"}`),
		found("repos/may-journal/fitness-runner/contents?ref="+refSha, `[{"name":"README.md"},{"name":"action.yml"}]`),
	})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "workflow-ref-exists", "1")
}

func Test0086_2(t *testing.T) {
	t.Parallel()
	out, code, _ := refCheck(t, refWorkflow(
		"may-journal/fitness-runner/.github/workflows/comment-check-reusable.yml@v1",
	), []response{
		found("repos/may-journal/fitness-runner/git/ref/tags/v1", `{"ref":"refs/tags/v1"}`),
		notFound("repos/may-journal/fitness-runner/contents/.github/workflows/comment-check-reusable.yml?ref=v1"),
		found("repos/may-journal/fitness-runner", `{"full_name":"may-journal/fitness-runner"}`),
	})
	sees(t, out, code, 1, ".github/workflows/ci.yml: line 5: may-journal/fitness-runner/.github/workflows/comment-check-reusable.yml@v1: "+
		".github/workflows/comment-check-reusable.yml does not exist at v1")
}

func Test0086_3(t *testing.T) {
	t.Parallel()
	out, code, _ := refCheck(t, refWorkflow("may-journal/fitness-runner@v9"), []response{
		notFound("repos/may-journal/fitness-runner/git/ref/tags/v9"),
		notFound("repos/may-journal/fitness-runner/git/ref/heads/v9"),
		found("repos/may-journal/fitness-runner", `{"full_name":"may-journal/fitness-runner"}`),
	})
	sees(t, out, code, 1, ".github/workflows/ci.yml: line 5: may-journal/fitness-runner@v9: no tag or branch v9 exists")
}

func Test0086_4(t *testing.T) {
	t.Parallel()
	out, code, rp := refCheck(t, refWorkflow(
		"./.github/workflows/local.yml",
		"actions/checkout@v4",
		"may-journal/fitness-runner/.github/workflows/ci-reusable.yml@${{ inputs.ref }}",
		"may-journal/fitness-runner/.github/workflows/ci-reusable.yml@main",
	), []response{
		notFound("repos/may-journal/fitness-runner/git/ref/tags/main"),
		found("repos/may-journal/fitness-runner/git/ref/heads/main", `{"ref":"refs/heads/main"}`),
	})
	sees(t, out, code, 0, "All 1 checks passed")
	if rp.called("gh", "contents") || rp.called("gh", "actions/checkout") {
		t.Errorf("an ignored line read GitHub: %q", rp.calls("gh"))
	}
}

func Test0086_5(t *testing.T) {
	t.Parallel()
	out, code, _ := refCheck(t, refWorkflow("may-journal/hub/.github/workflows/sync.yml@v1"), []response{
		notFound("repos/may-journal/hub"),
	})
	sees(t, out, code, 0, ".github/workflows/ci.yml: line 5: may-journal/hub/.github/workflows/sync.yml@v1: skipped, may-journal/hub cannot be read with this token")
}

func Test0086_6(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "workflow-ref-exists")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "workflow-ref-exists")
}
