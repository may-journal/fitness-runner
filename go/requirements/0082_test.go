package requirements

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sloppy is a comment whose first line is a change-log note.
const sloppy = "Update: the parser now handles tabs.\n"

// sloppyError is the one finding comment-check reports on sloppy.
const sloppyError = `(comment): line 1: "Update:": change-log note; edit the text in place and drop the note`

// commentURL is where comment 31, which every 0082 event names, lives.
const commentURL = "https://github.com/may-journal/example/issues/9#issuecomment-31"

// commentCheck runs comment-check the way its Actions step does, on the event
// payload GitHub would deliver, with gh answering from saved responses.
func commentCheck(t *testing.T, event string, payload map[string]any, saved []response, extra ...string) (string, int, replays) {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	path := filepath.Join(t.TempDir(), "event.json")
	data, err := json.Marshal(payload)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o644))
	rp := standIn(t, map[string][]response{"gh": saved})
	env := append(rp.env, "GITHUB_EVENT_NAME="+event, "GITHUB_EVENT_PATH="+path, "GITHUB_REPOSITORY=may-journal/example")
	out, code := fitness(t, repo, append(env, extra...), "comment-check")
	return out, code, rp
}

// eventComment is comment 31 with body, by login, an account of kind.
func eventComment(body, login, kind string) map[string]any {
	return map[string]any{"id": 31, "body": body, "html_url": commentURL, "user": map[string]any{"login": login, "type": kind}}
}

// judgeComment runs comment-check on comment 31 on issue 9, as posting it
// triggers it.
func judgeComment(t *testing.T, comment map[string]any, saved []response, extra ...string) (string, int, replays) {
	t.Helper()
	issue := map[string]any{"number": 9, "title": "t", "body": ""}
	return commentCheck(t, "issue_comment", map[string]any{"action": "created", "issue": issue, "comment": comment}, saved, extra...)
}

// commentGitHub is the saved GitHub holding notice, the bot's reply to
// comment 31 as the lookup prints it, and its thumbs-down, id 55. Every
// write succeeds.
func commentGitHub(notice string) []response {
	return []response{
		{Match: []string{"-X POST"}},
		{Match: []string{"-X PATCH"}},
		{Match: []string{"-X DELETE"}},
		{Match: []string{"reactions?content=-1"}, Stdout: "55\n"},
		{Match: []string{"fitness:comment-check:31:"}, Stdout: notice},
		{Match: []string{"fitness:comment-check:"}},
	}
}

// commentNotice is reply 41 stating verdict about comment 31.
func commentNotice(verdict string) string {
	sum := sha256.Sum256([]byte(verdict))
	mark := "<!-- fitness:comment-check:31:" + hex.EncodeToString(sum[:])[:12] + " -->"
	line, _ := json.Marshal(map[string]any{"id": 41, "body": "❌ reply\n\n" + mark})
	return string(line) + "\n"
}

// draft is a comment with a change-log note and a diagram whose table
// misses callout 2.
const draft = sloppy + "\n```mermaid\nflowchart TD\n  A[1 Start]:::s --> B[2 End]:::s\n  classDef s fill:#eef\n```\n\n" +
	"| # | Element | Description | Why |\n| - | - | - | - |\n| 1 | Start | First. | Begins. |\n"

func Test0082_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := fitness(t, repo, nil, "comment-check", "--blob", draft)
	sees(t, out, code, 1, sloppyError, "diagram callout 2 has no matching table row")
	out, code = piped(t, repo, draft, nil, "comment-check")
	sees(t, out, code, 1, sloppyError, "diagram callout 2 has no matching table row")
}

func Test0082_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := fitness(t, repo, nil, "comment-check", "--blob=The parser now handles tabs.\n\n"+numbered)
	sees(t, out, code, 0, "All 11 checks passed")
}

func Test0082_3(t *testing.T) {
	t.Parallel()
	out, code, rp := judgeComment(t, eventComment(sloppy, "dev", "User"), commentGitHub(""))
	sees(t, out, code, 1, "comment-check #9", "Update:", "Re-check a draft before posting with `fitness comment-check --blob -`")
	reply := rp.called("gh", "-X POST", "body=❌ @dev [This comment]("+commentURL+")", "Update:", "comment-check --blob -", "issues/9/comments")
	if !reply || !rp.called("gh", "-X POST", "content=-1", "issues/comments/31/reactions") {
		t.Errorf("want one reply naming the author and a thumbs-down, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_4(t *testing.T) {
	t.Parallel()
	out, code, rp := judgeComment(t, eventComment(sloppy, "dev", "User"), commentGitHub(commentNotice(sloppyError)))
	sees(t, out, code, 1, "Update:")
	if rp.called("gh", "body=") {
		t.Errorf("want no comment written, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_5(t *testing.T) {
	t.Parallel()
	fixed := eventComment("The parser now handles tabs.\n", "dev", "User")
	out, code, rp := judgeComment(t, fixed, commentGitHub(commentNotice(sloppyError)))
	sees(t, out, code, 0, "Passed.")
	edit := rp.called("gh", "-X PATCH", "body=✅ [This comment]("+commentURL+") now passes", "issues/comments/41")
	if !edit || !rp.called("gh", "-X DELETE", "issues/comments/31/reactions/55") {
		t.Errorf("want the reply turned into a passing note and the reaction removed, got gh calls %q", rp.calls("gh"))
	}
}

func Test0082_6(t *testing.T) {
	t.Parallel()
	verdict := "✅ Plan looks good.\n\n<!-- fitness:plan-structure:0123456789ab -->"
	for _, c := range []struct {
		comment map[string]any
		ignore  string
	}{
		{eventComment(sloppy, "may-journal-automation[bot]", "Bot"), ""},
		{eventComment(verdict, "dev", "User"), ""},
		{eventComment(sloppy, "someone", "User"), "release-bot[bot], someone"},
	} {
		out, code, rp := judgeComment(t, c.comment, commentGitHub(""), "FITNESS_COMMENT_IGNORE_AUTHORS="+c.ignore)
		sees(t, out, code, 0, "Skipped "+commentURL, "0 targets checked")
		if len(rp.calls("gh")) > 0 {
			t.Errorf("want nothing read or written, got gh calls %q", rp.calls("gh"))
		}
	}
}

func Test0082_7(t *testing.T) {
	t.Parallel()
	comment := eventComment(sloppy, "dev", "User")
	comment["in_reply_to_id"] = 30
	pr := map[string]any{"number": 9, "title": "t", "body": ""}
	payload := map[string]any{"action": "created", "pull_request": pr, "comment": comment}
	out, code, rp := commentCheck(t, "pull_request_review_comment", payload, commentGitHub(""))
	sees(t, out, code, 1, "Update:")
	reply := rp.called("gh", "-X POST", "body=❌ @dev", "pulls/9/comments/30/replies")
	if !reply || !rp.called("gh", "-X POST", "content=-1", "pulls/comments/31/reactions") {
		t.Errorf("want a reply in the review thread and a thumbs-down, got gh calls %q", rp.calls("gh"))
	}
}

// sweptComment is one comment as the repository comment list prints it.
func sweptComment(id int, parent string) string {
	line, _ := json.Marshal(map[string]any{"id": id, "body": sloppy, "url": commentURL, "author": "dev", "type": "User",
		"parent": "https://api.github.com/repos/may-journal/example/" + parent, "in_reply_to_id": 0})
	return string(line) + "\n"
}

func Test0082_8(t *testing.T) {
	t.Parallel()
	saved := append([]response{
		{Match: []string{"issues/comments?per_page=100&since="}, Stdout: sweptComment(31, "issues/9")},
		{Match: []string{"pulls/comments?per_page=100&since="}, Stdout: sweptComment(32, "pulls/9")},
	}, commentGitHub("")...)
	out, code, rp := commentCheck(t, "workflow_dispatch", map[string]any{}, saved)
	sees(t, out, code, 1, "2 targets checked")
	issue := rp.called("gh", "-X POST", "body=❌", "issues/9/comments")
	if !issue || !rp.called("gh", "-X POST", "body=❌", "pulls/9/comments/32/replies") {
		t.Errorf("want the issue and review comments judged, got gh calls %q", rp.calls("gh"))
	}
	since := strings.Contains(strings.Join(rp.calls("gh"), "\n"), "since=")
	if !since {
		t.Errorf("want the sweep limited to recent comments, got gh calls %q", rp.calls("gh"))
	}
}
