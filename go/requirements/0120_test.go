package requirements

import (
	"encoding/json"
	"strings"
	"testing"
)

// reviewComments is review 70's ten inline comments on PR 9; ids 31 and 32 are
// sloppy, the rest pass.
func reviewComments() string {
	var lines strings.Builder
	for id := 31; id <= 40; id++ {
		body := "The parser now handles tabs.\n"
		if id <= 32 {
			body = sloppy
		}
		line, _ := json.Marshal(map[string]any{"id": id, "body": body, "url": commentURL, "author": "dev",
			"parent": "https://api.github.com/repos/may-journal/example/pulls/9", "in_reply_to_id": 0})
		lines.Write(append(line, '\n'))
	}
	return lines.String()
}

func Test0120_1(t *testing.T) {
	t.Parallel()
	saved := append([]response{{Match: []string{"pulls/9/reviews/70/comments"}, Stdout: reviewComments()}}, commentGitHub("")...)
	pr := map[string]any{"number": 9, "title": "t", "body": ""}
	review := map[string]any{"id": 70, "user": map[string]any{"login": "dev", "type": "User"}}
	payload := map[string]any{"action": "submitted", "pull_request": pr, "review": review}
	out, code, rp := commentCheck(t, "pull_request_review", payload, saved)
	sees(t, out, code, 1, "10 targets checked")
	if rp.count("gh", "-X POST", "body=❌ @dev") != 2 {
		t.Errorf("want two replies, one per failing comment, got gh calls %q", rp.calls("gh"))
	}
	for _, root := range []string{"31", "32"} {
		if !rp.called("gh", "-X POST", "body=❌ @dev", "pulls/9/comments/"+root+"/replies") {
			t.Errorf("want a reply in comment %s's thread, got gh calls %q", root, rp.calls("gh"))
		}
	}
}
