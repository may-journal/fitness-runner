package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

func init() {
	register("auto-merge", "Turn on squash auto-merge for the event's PR, or merge the stack it tops, and note the result on each PR.", nil, autoMerge)
}

// pullEvent is the part of a pull_request event auto-merge reads.
type pullEvent struct {
	PullRequest struct {
		URL    string `json:"html_url"`
		Number int    `json:"number"`
		Base   struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
	Repository struct {
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
}

// readPullEvent reads the event that started the run.
func readPullEvent() (pullEvent, error) {
	var event pullEvent
	raw, err := os.ReadFile(env("GITHUB_EVENT_PATH"))
	if err == nil {
		err = json.Unmarshal(raw, &event)
	}
	return event, err
}

func autoMerge(actions.Step, []string) error {
	event, err := readPullEvent()
	if err != nil {
		return err
	}
	pr, def := pullRequest{strconv.Itoa(event.PullRequest.Number)}, event.Repository.DefaultBranch
	// Only a native stack's PRs carry a stack object.
	answer, err := actions.GhAnswer("GitHub refused to show this PR", "api", pr.path())
	var pull struct {
		Stack struct{ Number json.Number } `json:"stack"`
	}
	_ = json.Unmarshal([]byte(answer), &pull)
	switch base := event.PullRequest.Base.Ref; {
	case err != nil:
		return err
	case pull.Stack.Number != "":
		return mergeStack(string(pull.Stack.Number), def)
	case base != def:
		actions.Notice(fmt.Sprintf("This PR targets %s, not %s, so it gets no auto-merge; GitHub has none for branch chains. Its auto-merge comment says how to merge it.", base, def))
		say(pr.number, fmt.Sprintf("Auto-merge is off: this PR targets `%s`, not `%s`, in a branch chain. The bot cannot merge a chain: merging this PR into `%s` pushes to that PR and dismisses its approval. Merge it by hand, or rebuild the chain with `gh stack`, which the bot merges whole. Once GitHub moves this PR onto `%s`, auto-merge turns on and this comment says so.", base, def, base, def))
		return nil
	}
	return enable(event.PullRequest.URL, pr.number, def)
}

// enable turns on squash auto-merge for the PR at url, and updates its note
// when an earlier run left one.
func enable(url, number, def string) error {
	answer, err := actions.Call{}.Combined("gh", "pr", "merge", "--auto", "--squash", url)
	if answer != "" {
		fmt.Println(answer)
	}
	switch {
	case err == nil:
		sayIfNoted(number, fmt.Sprintf("Auto-merge is on: this PR squash-merges into `%s` once it is approved and green.", def))
		return nil
	case strings.Contains(strings.ToLower(answer), "auto-merge is not supported for stacked"):
		actions.Notice("GitHub still counts this PR as stacked, so it gets no auto-merge. Its auto-merge comment says how to merge it.")
		say(number, fmt.Sprintf("Auto-merge is off: this PR targets `%s`, but GitHub still counts it as part of a stack and refuses auto-merge. Merge it by hand once it is approved and green, or push to it once GitHub moves it out of the stack.", def))
		return nil
	}
	return actions.Fail("GitHub refused auto-merge for this PR: " + answer)
}

// stack is a native stack's open PRs and the top one it merges through.
type stack struct {
	number, top, sha string
	members          []string
	// list names the members as `#2 #3`.
	list string
}

// stackPR is one PR as GitHub lists it in a stack.
type stackPR struct {
	Number int
	State  string
	Head   struct{ Ref, SHA string }
	Base   struct{ Ref string }
}

// mergeStack merges stack number through its top PR once every open PR in
// it is approved, green and mergeable.
func mergeStack(number, def string) error {
	s, err := readStack(number)
	if err != nil {
		return err
	}
	if waits, err := stackWaits(s.members); err != nil || waits != "" {
		return sayWaiting(s, def, waits, err)
	}
	answer, err := requestMerge(s)
	if err != nil {
		return err
	}
	return finishMerge(s, def, answer)
}

// requestMerge asks GitHub to merge the stack through its top PR, and
// returns its answer once the merge is no longer pending.
func requestMerge(s stack) (string, error) {
	answer, err := actions.Call{}.Combined("gh", "api", "-X", "PUT", actions.RepoPath()+"/pulls/"+s.top+"/merge-async", "-f", "merge_method=squash", "-f", "sha="+s.sha, "-F", "bypass_rules=false")
	fmt.Println(answer)
	// A 409 names the merge already running, which the job polls.
	if merge := mergeAnswer(answer); err != nil && merge.Details.UUID == "" {
		say(s.top, fmt.Sprintf("Auto-merge could not merge stack #%s (%s): GitHub refused, %s.", s.number, s.list, orNoReason(merge.Details.Message)))
		return answer, actions.Fail("GitHub refused to merge stack #" + s.number + " through #" + s.top + ": " + answer)
	}
	return pollMerge(s, answer)
}

// readStack lists stack number's open PRs and finds its one top PR, the one
// no other builds on; merged PRs drop out.
func readStack(number string) (stack, error) {
	answer, err := actions.GhAnswer("GitHub refused to show stack #"+number, "api", actions.RepoPath()+"/stacks/"+number)
	var listed struct {
		PullRequests []stackPR `json:"pull_requests"`
	}
	if err != nil || json.Unmarshal([]byte(answer), &listed) != nil {
		return stack{}, cmpOrErr(err, "GitHub's stack answer is not JSON: "+answer)
	}
	open := slices.DeleteFunc(listed.PullRequests, func(pr stackPR) bool { return pr.State != "open" })
	tops := slices.DeleteFunc(slices.Clone(open), func(pr stackPR) bool {
		return slices.ContainsFunc(open, func(o stackPR) bool { return o.Base.Ref == pr.Head.Ref })
	})
	if len(open) == 0 || len(tops) != 1 {
		return stack{}, actions.Fail("Stack #" + number + " has no single open top PR: " + answer)
	}
	return newStack(number, open, tops[0]), nil
}

// newStack names stack number's open members and its top PR.
func newStack(number string, open []stackPR, top stackPR) stack {
	s := stack{number: number, top: strconv.Itoa(top.Number), sha: top.Head.SHA}
	for _, pr := range open {
		s.members = append(s.members, strconv.Itoa(pr.Number))
	}
	s.list = "#" + strings.Join(s.members, " #")
	return s
}

// cmpOrErr is err, or a failure naming problem when err is nil.
func cmpOrErr(err error, problem string) error {
	if err != nil {
		return err
	}
	return actions.Fail(problem)
}

// stackWaits names each member that cannot merge yet, as ` #2 is a draft;
// #3 needs approval`, or "".
func stackWaits(members []string) (string, error) {
	var waits []string
	for _, pr := range members {
		answer, err := actions.GhAnswer("GitHub refused to show #"+pr, "pr", "view", pr, "--repo", env("GITHUB_REPOSITORY"), "--json", "isDraft,reviewDecision,mergeable,statusCheckRollup")
		var state prState
		if err == nil {
			err = json.Unmarshal([]byte(answer), &state)
		}
		if err != nil {
			return "", err
		}
		if why := state.why(); why != "" {
			waits = append(waits, " #"+pr+" "+why)
		}
	}
	return strings.Join(waits, ";"), nil
}

// sayWaiting notes on every member what the stack waits for, or returns err.
func sayWaiting(s stack, def, waiting string, err error) error {
	if err != nil {
		return err
	}
	actions.Notice(fmt.Sprintf("Stack #%s waits:%s.", s.number, waiting))
	sayAll(s.members, fmt.Sprintf("Auto-merge waits: stack #%s (%s) merges into `%s` once every PR in it is approved, green and mergeable. Waiting:%s.", s.number, s.list, def, waiting))
	return nil
}

// pollMerge reads a pending merge every five seconds, 36 times at most, and
// returns GitHub's last answer. A 409's body names the running merge as
// pending, so a second run polls the first run's merge.
func pollMerge(s stack, answer string) (string, error) {
	uuid := mergeAnswer(answer).Details.UUID
	for tries := 0; mergeAnswer(answer).Status == "pending" && tries < 36; tries++ {
		time.Sleep(time.Duration(min(tries, 1)) * 5 * time.Second)
		var err error
		if answer, err = actions.GhAnswer("GitHub refused to show the merge of stack #"+s.number, "api", actions.RepoPath()+"/pulls/"+s.top+"/merge-async/"+uuid); err != nil {
			return answer, err
		}
	}
	return answer, nil
}

// finishMerge notes how GitHub's merge ended.
func finishMerge(s stack, def, answer string) error {
	merge := mergeAnswer(answer)
	switch merge.Status {
	case "merged", "enqueued":
		actions.Notice(fmt.Sprintf("GitHub %s stack #%s through #%s.", merge.Status, s.number, s.top))
		sayAll(s.members, fmt.Sprintf("Auto-merge %s stack #%s (%s) into `%s` through #%s.", merge.Status, s.number, s.list, def, s.top))
		return nil
	case "pending":
		actions.Warning(fmt.Sprintf("GitHub is still merging stack #%s through #%s after three minutes.", s.number, s.top))
		say(s.top, fmt.Sprintf("Auto-merge asked GitHub to merge stack #%s (%s) through #%s, and GitHub was still merging three minutes later. This PR shows the result.", s.number, s.list, s.top))
		return nil
	}
	say(s.top, fmt.Sprintf("Auto-merge could not merge stack #%s (%s): GitHub failed, %s.", s.number, s.list, orNoReason(merge.Details.Message)))
	return actions.Fail("GitHub failed to merge stack #" + s.number + " through #" + s.top + ": " + answer)
}

// mergeResult is GitHub's answer to a stack merge or a poll of one.
type mergeResult struct {
	Status  string
	Details struct{ UUID, Message string }
}

// mergeAnswer reads GitHub's JSON answer, dropping the CLI's own error line
// after it, or returns an empty result.
func mergeAnswer(answer string) mergeResult {
	body, _, _ := strings.Cut(answer, "\ngh: ")
	var result mergeResult
	_ = json.Unmarshal([]byte(body), &result)
	return result
}

// orNoReason is message, or a note that GitHub gave none.
func orNoReason(message string) string {
	return cmpOr(message, "giving no reason")
}

// prState is the part of a PR view that says whether it can merge.
type prState struct {
	IsDraft           bool
	ReviewDecision    string
	Mergeable         string
	StatusCheckRollup []struct{ WorkflowName, State, Status, Conclusion string }
}

// why says why the PR cannot merge yet, or "".
func (pr prState) why() string {
	switch {
	case pr.IsDraft:
		return "is a draft"
	case !slices.Contains([]string{"", "APPROVED"}, pr.ReviewDecision):
		return "needs approval"
	case pr.Mergeable == "CONFLICTING":
		return "has conflicts"
	case slices.ContainsFunc(pr.StatusCheckRollup, failing):
		return "is not green"
	}
	return ""
}

// failing reports whether a check other than Auto-merge's own, still running
// now, has not passed; GitHub's merge rechecks every rule itself.
func failing(c struct{ WorkflowName, State, Status, Conclusion string }) bool {
	passed := c.State == "SUCCESS" || c.Status == "COMPLETED" && slices.Contains([]string{"SUCCESS", "NEUTRAL", "SKIPPED"}, c.Conclusion)
	return !passed && !strings.HasPrefix(c.WorkflowName, "Auto-merge")
}
