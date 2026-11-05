// cspell:ignore fbody
package requirements

import "testing"

// redComment is a comment markdown-no-bold-italic fails.
const redComment = "Some **bold** text."

// denied asserts the hook denies each command, naming the bold finding, in
// its own run named for the command.
func denied(t *testing.T, repo string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			out, code := hookEvent(t, repo, bashEvent(command))
			sees(t, out, code, 0, `"permissionDecision":"deny"`, "[markdown-no-bold-italic]")
		})
	}
}

// passed asserts the hook passes each command silently, in its own run
// named for the command.
func passed(t *testing.T, repo string, commands ...string) {
	t.Helper()
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			out, code := hookEvent(t, repo, bashEvent(command))
			silentPass(t, out, code)
		})
	}
}

func Test0145_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"reply.md": redComment + "\n"})
	denied(t, repo,
		"gh api repos/o/r/issues/5/comments -f body='"+redComment+"'",
		"gh api -X PATCH repos/{owner}/{repo}/issues/comments/9 --raw-field body='"+redComment+"'",
		"gh api repos/o/r/pulls/5/comments -f path=a.go -F line=3 -F body=@reply.md",
		"gh api repos/o/r/pulls/comments/9/replies -fbody='"+redComment+"'",
		"gh api https://api.github.com/repos/o/r/commits/abc/comments -f body='"+redComment+"'",
		"gh api repos/o/r/pulls/5/reviews/9/events -f event=COMMENT -f body='"+redComment+"'",
	)
}

func Test0145_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	review := `{"event": "COMMENT", "body": "Fine.", "comments": [{"path": "a.go", "body": "Fine."}, {"path": "b.go", "body": "` + redComment + `"}]}`
	write(t, repo, map[string]string{"review.json": review + "\n"})
	out, code := hookEvent(t, repo, bashEvent("gh api repos/o/r/pulls/5/reviews --input review.json"))
	sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on review line comment 2")
	fields := "gh api repos/o/r/pulls/5/reviews -f event=COMMENT -f body=Fine. -f 'comments[][path]=a.go' -f 'comments[][body]="
	out, code = hookEvent(t, repo, bashEvent(fields+redComment+"'"))
	sees(t, out, code, 0, `"permissionDecision":"deny"`, "failed on review line comment 1")
	out, code = hookEvent(t, repo, bashEvent(fields+"Fine.'"))
	silentPass(t, out, code)
}

func Test0145_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	mutation := `mutation($id: ID!, $body: String!) { addComment(input: {subjectId: $id, body: $body}) { clientMutationId } }`
	thread := `mutation($id: ID!, $text: String!) { addPullRequestReviewThread(input: {pullRequestId: $id, path: "a.go", line: 3, body: $text}) { clientMutationId } }`
	request := `{"query": "mutation($b: String!) { addDiscussionComment(input: {discussionId: \"D\", body: $b}) { clientMutationId } }", "variables": {"b": "` + redComment + `"}}`
	write(t, repo, map[string]string{"request.json": request + "\n"})
	denied(t, repo,
		"gh api graphql -f query='"+mutation+"' -f id=X -f body='"+redComment+"'",
		"gh api graphql -f query='"+thread+"' -f id=X -f text='"+redComment+"'",
		"gh api graphql --input request.json",
		"gh issue close 5 --comment '"+redComment+"'",
		"gh pr close 5 -c '"+redComment+"'",
	)
}

func Test0145_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, map[string]string{"reply.md": redComment + "\n"})
	passed(t, repo,
		"gh api repos/o/r/pulls/5/comments",
		"gh api -X GET repos/o/r/issues/5/comments -f per_page=100",
		`gh api repos/o/r/issues/5/comments -f body="$BODY"`,
		`gh api repos/o/r/issues/5/comments -F "body=@${DIR}reply.md"`,
		"gh api graphql -f query='query { viewer { login } }'",
		"gh api repos/o/r/issues/5/labels -f 'labels[]=bug'",
	)
}
