package main

import (
	"strings"
	"testing"
)

const openChecklist = "## What needs to happen\n\n- [x] done\n- [ ] write docs\n- [ ] ship it\n"

// closedEvent is an issues: closed payload for issue 8, closed by sender.
func closedEvent(body, reason, sender string) ghEvent {
	ev := ghEvent{name: "issues", Action: "closed", Issue: &target{
		Number: 8, Body: body, StateReason: reason, ClosedAt: "2026-09-28T12:00:00Z",
	}}
	ev.Sender.Login = sender
	return ev
}

func TestCloseCheckReopensAManualClose(t *testing.T) {
	gh := &fakeGH{}
	ev := closedEvent(openChecklist, "completed", "alice")
	if code := closeCheck(ev, gh); code != 0 {
		t.Fatalf("closeCheck = %d, want 0", code)
	}
	if len(gh.reopened) != 1 || gh.reopened[0] != 8 {
		t.Fatalf("reopened = %v, want [8]", gh.reopened)
	}
	got := gh.comments[8]
	if len(got) != 1 {
		t.Fatalf("comments = %d, want 1", len(got))
	}
	for _, want := range []string{"- write docs\n- ship it\n", "cc @alice\n", "<!-- fitness:close-check:"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("comment missing %q: %q", want, got[0])
		}
	}
	if strings.Contains(got[0], "- done") {
		t.Errorf("comment lists a ticked item: %q", got[0])
	}
	closeCheck(ev, gh)
	if len(gh.comments[8]) != 1 {
		t.Errorf("comments after re-run = %d, want 1 (the re-run stays quiet)", len(gh.comments[8]))
	}
}

func TestCloseCheckMentionsTheClosingPRsAuthorAndMerger(t *testing.T) {
	gh := &fakeGH{closers: map[int]closer{8: {Author: "bob", Merger: "carol"}}}
	closeCheck(closedEvent(openChecklist, "completed", "carol"), gh)
	if len(gh.reopened) != 1 {
		t.Fatalf("reopened = %v, want one reopen", gh.reopened)
	}
	if !strings.Contains(gh.comments[8][0], "cc @carol @bob\n") {
		t.Errorf("comment = %q, want cc @carol @bob, each once", gh.comments[8][0])
	}
}

func TestCloseCheckSkipsAppMentions(t *testing.T) {
	gh := &fakeGH{closers: map[int]closer{8: {Author: "bob", Merger: "github-actions[bot]"}}}
	closeCheck(closedEvent(openChecklist, "", "github-actions[bot]"), gh)
	if !strings.Contains(gh.comments[8][0], "cc @bob\n") {
		t.Errorf("comment = %q, want cc @bob only", gh.comments[8][0])
	}
}

func TestCloseCheckLeavesDroppedWorkClosed(t *testing.T) {
	for _, reason := range []string{"not_planned", "duplicate"} {
		gh := &fakeGH{}
		if code := closeCheck(closedEvent(openChecklist, reason, "alice"), gh); code != 0 {
			t.Fatalf("%s: closeCheck = %d, want 0", reason, code)
		}
		if len(gh.reopened) != 0 || len(gh.comments) != 0 {
			t.Errorf("%s: reopened %v, comments %v; want neither", reason, gh.reopened, gh.comments)
		}
	}
}

func TestCloseCheckIgnoresFencedAndTickedBoxes(t *testing.T) {
	for name, body := range map[string]string{
		"fenced": "- [x] done\n```md\n- [ ] an example in a code block\n```\n",
		"ticked": "- [x] one\n- [X] two\n",
		"none":   "no checklist at all",
	} {
		gh := &fakeGH{}
		if code := closeCheck(closedEvent(body, "completed", "alice"), gh); code != 0 {
			t.Fatalf("%s: closeCheck = %d, want 0", name, code)
		}
		if len(gh.reopened) != 0 || len(gh.comments) != 0 {
			t.Errorf("%s: reopened %v, comments %v; want neither", name, gh.reopened, gh.comments)
		}
	}
}

func TestCloseCheckNeedsAClosedEvent(t *testing.T) {
	ev := closedEvent(openChecklist, "completed", "alice")
	ev.Action = "reopened"
	if code := closeCheck(ev, &fakeGH{}); code != 1 {
		t.Fatalf("closeCheck on %q = %d, want 1", ev.Action, code)
	}
	if code := closeCheck(ghEvent{name: "workflow_dispatch"}, &fakeGH{}); code != 1 {
		t.Fatalf("closeCheck with no issue = %d, want 1", code)
	}
}
