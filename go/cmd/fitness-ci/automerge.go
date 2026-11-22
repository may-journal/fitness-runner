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

// autoMergeMarker ends the bot's note, so a later run finds and edits it.
const autoMergeMarker = "<!-- fitness:auto-merge -->"

// autoMergeRefused is the warning when GitHub refuses a change to a note.
const autoMergeRefused = "GitHub refused a change to a PR's auto-merge comment; its answer is above."

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

// merger turns on auto-merge in one repo, keeping one note per PR.
type merger struct {
	repo, bot string
	// noted is the PR whose note the last lookup found as id, or "".
	noted, id string
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
	m := &merger{repo: "repos/" + env("GITHUB_REPOSITORY"), bot: env("BOT")}
	number, def := strconv.Itoa(event.PullRequest.Number), event.Repository.DefaultBranch
	stack, err := m.stackOf(number)
	if err != nil || stack != "" {
		return m.mergeStackOr(err, stack, def)
	}
	if base := event.PullRequest.Base.Ref; base != def {
		return m.skipChain(number, base, def)
	}
	return m.enable(event.PullRequest.URL, number, def)
}

// mergeStackOr returns err, or merges the stack when there is no error.
func (m *merger) mergeStackOr(err error, stack, def string) error {
	if err != nil {
		return err
	}
	return m.mergeStack(stack, def)
}

// stackOf is the native stack PR number belongs to, or "" for none. Only a
// native stack's PRs carry a stack object.
func (m *merger) stackOf(number string) (string, error) {
	answer, err := actions.Call{}.Combined("gh", "api", m.repo+"/pulls/"+number)
	if err != nil {
		return "", failAnswer("GitHub refused to show this PR", answer)
	}
	var pull struct {
		Stack struct {
			Number json.Number `json:"number"`
		} `json:"stack"`
	}
	_ = json.Unmarshal([]byte(answer), &pull)
	return string(pull.Stack.Number), nil
}

// skipChain notes that a PR based on another branch gets no auto-merge.
func (m *merger) skipChain(number, base, def string) error {
	actions.Notice(fmt.Sprintf("This PR targets %s, not %s, so it gets no auto-merge; GitHub has none for branch chains. Its auto-merge comment says how to merge it.", base, def))
	m.say(number, fmt.Sprintf("Auto-merge is off: this PR targets `%s`, not `%s`, in a branch chain. The bot cannot merge a chain: merging this PR into `%s` pushes to that PR and dismisses its approval. Merge it by hand, or rebuild the chain with `gh stack`, which the bot merges whole. Once GitHub moves this PR onto `%s`, auto-merge turns on and this comment says so.", base, def, base, def))
	return nil
}

// enable turns on squash auto-merge for the PR at url.
func (m *merger) enable(url, number, def string) error {
	answer, err := actions.Call{}.Combined("gh", "pr", "merge", "--auto", "--squash", url)
	if answer != "" {
		fmt.Println(answer)
	}
	if err == nil {
		m.noteEnabled(number, def)
		return nil
	}
	if strings.Contains(strings.ToLower(answer), "auto-merge is not supported for stacked") {
		actions.Notice("GitHub still counts this PR as stacked, so it gets no auto-merge. Its auto-merge comment says how to merge it.")
		m.say(number, fmt.Sprintf("Auto-merge is off: this PR targets `%s`, but GitHub still counts it as part of a stack and refuses auto-merge. Merge it by hand once it is approved and green, or push to it once GitHub moves it out of the stack.", def))
		return nil
	}
	return failAnswer("GitHub refused auto-merge for this PR", answer)
}

// noteEnabled updates an earlier note to say auto-merge is on; a PR with no
// note gets none.
func (m *merger) noteEnabled(number, def string) {
	if m.note(number) != nil {
		actions.Warning(autoMergeRefused)
	} else if m.id != "" {
		m.say(number, fmt.Sprintf("Auto-merge is on: this PR squash-merges into `%s` once it is approved and green.", def))
	}
}

// stackPR is one PR as GitHub lists it in a stack.
type stackPR struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	Head   struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

// stackInfo is a stack's open PRs and the top one it merges through.
type stackInfo struct {
	number, top, sha string
	members          []string
	// list names the members as `#2 #3`.
	list string
}

// mergeStack merges stack through its top PR once every open PR in it is
// ready.
func (m *merger) mergeStack(number, def string) error {
	s, err := readStack(m.repo, number)
	if err != nil {
		return err
	}
	waits, err := stackWaits(s.members)
	if err != nil {
		return err
	}
	if len(waits) > 0 {
		m.sayWaiting(s, def, strings.Join(waits, ";"))
		return nil
	}
	merge, answer, err := m.requestMerge(s)
	if err != nil {
		return err
	}
	return m.finishMerge(s, def, merge, answer)
}

// readStack lists stack number's open PRs and finds its one top PR.
func readStack(repo, number string) (stackInfo, error) {
	answer, err := actions.Call{}.Combined("gh", "api", repo+"/stacks/"+number)
	if err != nil {
		return stackInfo{}, failAnswer("GitHub refused to show stack #"+number, answer)
	}
	var listed struct {
		PullRequests []stackPR `json:"pull_requests"`
	}
	if err := json.Unmarshal([]byte(answer), &listed); err != nil {
		return stackInfo{}, err
	}
	// Merged PRs drop out; the top PR is the one no other builds on.
	open := openPRs(listed.PullRequests)
	tops := topPRs(open)
	if len(open) == 0 || len(tops) != 1 {
		return stackInfo{}, failAnswer("Stack #"+number+" has no single open top PR", answer)
	}
	return newStackInfo(number, open, tops[0]), nil
}

// openPRs is the open PRs among prs.
func openPRs(prs []stackPR) []stackPR {
	var open []stackPR
	for _, pr := range prs {
		if pr.State == "open" {
			open = append(open, pr)
		}
	}
	return open
}

// topPRs is the PRs of open that no other PR of open builds on.
func topPRs(open []stackPR) []stackPR {
	var tops []stackPR
	for _, pr := range open {
		if !slices.ContainsFunc(open, func(o stackPR) bool { return o.Base.Ref == pr.Head.Ref }) {
			tops = append(tops, pr)
		}
	}
	return tops
}

// newStackInfo names stack number's members and its top PR.
func newStackInfo(number string, open []stackPR, top stackPR) stackInfo {
	s := stackInfo{number: number, top: strconv.Itoa(top.Number), sha: top.Head.SHA}
	tags := make([]string, len(open))
	for i, pr := range open {
		s.members = append(s.members, strconv.Itoa(pr.Number))
		tags[i] = "#" + s.members[i]
	}
	s.list = strings.Join(tags, " ")
	return s
}

// stackWaits lists each member that cannot merge yet, as ` #2 is a draft`.
func stackWaits(members []string) ([]string, error) {
	var waits []string
	for _, pr := range members {
		answer, err := actions.Call{}.Combined("gh", "pr", "view", pr, "--repo", env("GITHUB_REPOSITORY"), "--json", "isDraft,reviewDecision,mergeable,statusCheckRollup")
		if err != nil {
			return nil, failAnswer("GitHub refused to show #"+pr, answer)
		}
		var state prState
		if err := json.Unmarshal([]byte(answer), &state); err != nil {
			return nil, err
		}
		if why := state.why(); why != "" {
			waits = append(waits, " #"+pr+" "+why)
		}
	}
	return waits, nil
}

// sayWaiting notes on every member what the stack waits for.
func (m *merger) sayWaiting(s stackInfo, def, waiting string) {
	actions.Notice(fmt.Sprintf("Stack #%s waits:%s.", s.number, waiting))
	for _, pr := range s.members {
		m.say(pr, fmt.Sprintf("Auto-merge waits: stack #%s (%s) merges into `%s` once every PR in it is approved, green and mergeable. Waiting:%s.", s.number, s.list, def, waiting))
	}
}

// requestMerge asks GitHub to merge the stack through its top PR, then polls
// the merge for three minutes while it is pending.
func (m *merger) requestMerge(s stackInfo) (mergeResult, string, error) {
	answer, err := actions.Call{}.Combined("gh", "api", "-X", "PUT", m.repo+"/pulls/"+s.top+"/merge-async", "-f", "merge_method=squash", "-f", "sha="+s.sha, "-F", "bypass_rules=false")
	fmt.Println(answer)
	merge := mergeAnswer(answer)
	// A 409 names the merge already running, which the job polls.
	if err != nil && merge.Details.UUID == "" {
		m.say(s.top, fmt.Sprintf("Auto-merge could not merge stack #%s (%s): GitHub refused, %s.", s.number, s.list, orNoReason(merge.Details.Message)))
		return merge, answer, failAnswer("GitHub refused to merge stack #"+s.number+" through #"+s.top, answer)
	}
	return m.pollMerge(s, merge, answer)
}

// pollMerge reads a pending merge every five seconds, 36 times at most. A
// 409's body names the running merge as pending, so a second run polls the
// first run's merge.
func (m *merger) pollMerge(s stackInfo, merge mergeResult, answer string) (mergeResult, string, error) {
	uuid := merge.Details.UUID
	for tries := 0; merge.Status == "pending" && tries < 36; tries++ {
		if tries > 0 {
			time.Sleep(5 * time.Second)
		}
		var err error
		if answer, err = (actions.Call{}).Combined("gh", "api", m.repo+"/pulls/"+s.top+"/merge-async/"+uuid); err != nil {
			return merge, answer, failAnswer("GitHub refused to show the merge of stack #"+s.number, answer)
		}
		merge = mergeAnswer(answer)
	}
	return merge, answer, nil
}

// finishMerge notes how GitHub's merge ended.
func (m *merger) finishMerge(s stackInfo, def string, merge mergeResult, answer string) error {
	switch merge.Status {
	case "merged", "enqueued":
		actions.Notice(fmt.Sprintf("GitHub %s stack #%s through #%s.", merge.Status, s.number, s.top))
		for _, pr := range s.members {
			m.say(pr, fmt.Sprintf("Auto-merge %s stack #%s (%s) into `%s` through #%s.", merge.Status, s.number, s.list, def, s.top))
		}
		return nil
	case "pending":
		actions.Warning(fmt.Sprintf("GitHub is still merging stack #%s through #%s after three minutes.", s.number, s.top))
		m.say(s.top, fmt.Sprintf("Auto-merge asked GitHub to merge stack #%s (%s) through #%s, and GitHub was still merging three minutes later. This PR shows the result.", s.number, s.list, s.top))
		return nil
	}
	m.say(s.top, fmt.Sprintf("Auto-merge could not merge stack #%s (%s): GitHub failed, %s.", s.number, s.list, orNoReason(merge.Details.Message)))
	return failAnswer("GitHub failed to merge stack #"+s.number+" through #"+s.top, answer)
}

// mergeResult is GitHub's answer to a stack merge or a poll of one.
type mergeResult struct {
	Status  string `json:"status"`
	Details struct {
		UUID    string `json:"uuid"`
		Message string `json:"message"`
	} `json:"details"`
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
	if message == "" {
		return "giving no reason"
	}
	return message
}

// prState is the part of a PR view that says whether it can merge.
type prState struct {
	IsDraft           bool          `json:"isDraft"`
	ReviewDecision    string        `json:"reviewDecision"`
	Mergeable         string        `json:"mergeable"`
	StatusCheckRollup []rollupCheck `json:"statusCheckRollup"`
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
	case !pr.green():
		return "is not green"
	}
	return ""
}

// green reports whether every check passed but Auto-merge's own runs, which
// are still running now; GitHub's merge rechecks every rule itself.
func (pr prState) green() bool {
	return !slices.ContainsFunc(pr.StatusCheckRollup, func(c rollupCheck) bool {
		return !strings.HasPrefix(c.WorkflowName, "Auto-merge") && !c.passed()
	})
}

// rollupCheck is one check or status in a PR's status rollup.
type rollupCheck struct {
	WorkflowName string `json:"workflowName"`
	State        string `json:"state"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
}

// passed reports whether a status succeeded or a check run completed well.
func (c rollupCheck) passed() bool {
	good := slices.Contains([]string{"SUCCESS", "NEUTRAL", "SKIPPED"}, c.Conclusion)
	return c.State == "SUCCESS" || c.Status == "COMPLETED" && good
}

// note finds the bot's note on PR pr, leaving its id, or "" for none.
func (m *merger) note(pr string) error {
	query := fmt.Sprintf(`.[] | select(.user.login == "%s" and (.body | contains("%s"))) | .id`, m.bot, autoMergeMarker)
	ids, err := actions.Gh("api", m.repo+"/issues/"+pr+"/comments", "--paginate", "--jq", query)
	if err != nil {
		return err
	}
	m.id, _, _ = strings.Cut(ids, "\n")
	m.noted = pr
	return nil
}

// noteOnce looks up PR pr's note unless the last lookup already did.
func (m *merger) noteOnce(pr string) error {
	if m.noted == pr {
		return nil
	}
	return m.note(pr)
}

// write posts the note text on PR pr, or edits it when its text changed.
// GitHub may store a body with CRLF line ends, so those never count.
func (m *merger) write(pr, text string) error {
	body := text + "\n\n" + autoMergeMarker
	if err := m.noteOnce(pr); err != nil {
		return err
	}
	m.noted = ""
	issues := m.repo + "/issues"
	if m.id == "" {
		_, err := actions.Gh("api", issues+"/"+pr+"/comments", "-f", "body="+body)
		return err
	}
	current, err := actions.Gh("api", issues+"/comments/"+m.id, "--jq", ".body")
	if err == nil && strings.ReplaceAll(current, "\r", "") != body {
		_, err = actions.Gh("api", "-X", "PATCH", issues+"/comments/"+m.id, "-f", "body="+body)
	}
	return err
}

// say writes the note text on PR pr, and warns without failing when GitHub
// refuses.
func (m *merger) say(pr, text string) {
	if m.write(pr, text) != nil {
		actions.Warning(autoMergeRefused)
	}
}

// failAnswer ends the job red with an error naming GitHub's answer.
func failAnswer(message, answer string) error {
	return actions.Fail(message + ": " + actions.Escape(answer))
}
