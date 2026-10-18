// Package metadata shares GitHub judgments between local checks and Actions.
package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// ghCommand declares one GitHub command: which targets an event names, the
// checks that judge each target's body blob, and the after-effect, named by
// code reference, that reacts on GitHub. runGH runs every declaration the
// same way, so a command adds no plumbing of its own.
type ghCommand struct {
	// kind is the URL segment for target links: "issues" or "pull".
	kind    string
	targets func(ev ghEvent, gh aftereffect.Client, b blobChecker) ([]target, error)
	checks  []string
	// kinds, when set, judges each target by its issue kind instead of
	// checks: the kinds the checkout's templates and kindRules name.
	kinds func(root string) map[string]issueKind
	// links judges what a body points at, such as the issues a PR closes.
	links  func(t target, gh aftereffect.Client, b blobChecker) []string
	effect aftereffect.Effect
	// failOnFindings exits 1 when any target fails; close-check instead
	// succeeds once its effect has reopened the issue.
	failOnFindings bool
	// hint ends a run summary with findings, such as how to re-check locally.
	hint string
}

// ghCommands are the GitHub commands, by subcommand name.
var ghCommands = map[string]ghCommand{
	"issue-check": {
		kind: "issues", targets: issueTargets, kinds: discoverKinds,
		effect: aftereffect.IssueVerdict, failOnFindings: true,
	},
	// plan-check is issue-check for Plan issues only, for repos whose
	// workflow still calls it.
	"plan-check": {
		kind: "issues", targets: planTargets, kinds: onlyKind("Plan"),
		effect: aftereffect.IssueVerdict, failOnFindings: true,
	},
	"pr-check": {
		kind: "pull", targets: prTargets,
		checks: append([]string{"pr-structure"}, docChecks...),
		links:  prLinks, failOnFindings: true,
	},
	"close-check": {
		kind: "issues", targets: closeTargets,
		checks: []string{"issue-checklist"},
		effect: aftereffect.ReopenUnfinished,
	},
	"comment-check": {
		kind: "issues", targets: commentTargets,
		checks: CommentChecks,
		effect: aftereffect.CommentReply, failOnFindings: true,
		hint: aftereffect.RecheckHint,
	},
}

// RunGH runs the named GitHub command against the Actions event.
func RunGH(name string) int {
	root, err := os.Getwd()
	if err != nil {
		return ghFailure(name, err)
	}
	ev, err := readEvent()
	if err != nil {
		return ghFailure(name, err)
	}
	gh, err := aftereffect.NewClient()
	if err != nil {
		return ghFailure(name, err)
	}
	return ghCommands[name].run(name, ev, gh, blobChecker{root})
}

// run judges each target, applies the effect, and reports the sweep.
func (c ghCommand) run(name string, ev ghEvent, gh aftereffect.Client, b blobChecker) int {
	targets, err := c.targets(ev, gh, b)
	if err != nil {
		return ghFailure(name, err)
	}
	kinds := c.kindsAt(b.root)
	rep := newTargetReport(name, c.kind)
	rep.hint = c.hint
	failed := false
	for _, t := range targets {
		bad, err := c.check(gh, b, rep, t, kinds)
		failed = failed || bad
		if err != nil {
			rep.finish()
			return ghFailure(fmt.Sprintf("%s #%d", name, t.Number), err)
		}
	}
	rep.finish()
	return c.exitCode(name, failed)
}

// kindsAt is the issue kinds the checkout at root names, or none when the
// command does not judge kinds.
func (c ghCommand) kindsAt(root string) map[string]issueKind {
	if c.kinds == nil {
		return nil
	}
	return c.kinds(root)
}

// check judges one target, reacts, and reports whether it failed. A target
// that left the repository, such as an issue moved to another repo, is
// noted and does not fail, as is an issue with no kind label when the
// command judges kinds.
func (c ghCommand) check(gh aftereffect.Client, b blobChecker, rep *targetReport, t target, kinds map[string]issueKind) (bool, error) {
	if t.Gone {
		rep.gone(t.Number)
		return false, nil
	}
	t, errs, skip := c.findings(gh, b, t, kinds)
	if skip != "" {
		rep.note(t.Number, skip)
		return false, nil
	}
	err := c.react(gh, t, errs)
	if errors.Is(err, aftereffect.ErrIssueGone) {
		rep.gone(t.Number)
		return false, nil
	}
	rep.add(t.Number, errs)
	return len(errs) > 0, err
}

// findings judges t by the declared checks, or by its issue kind when the
// command judges kinds, returning t with its kind set. skip says why an
// issue with no kind label is not checked.
func (c ghCommand) findings(gh aftereffect.Client, b blobChecker, t target, kinds map[string]issueKind) (target, []string, string) {
	if c.kinds == nil {
		return t, c.judge(t, gh, b), ""
	}
	k, labelErrs, ok := classify(t, kinds)
	if !ok {
		return t, nil, fmt.Sprintf("#%d has no issue kind label, so it is not checked.", t.Number)
	}
	t.Kind = k.label
	return t, append(labelErrs, k.judge(t, gh, b)...), ""
}

// judge runs the declared checks on the target's body and its links.
func (c ghCommand) judge(t target, gh aftereffect.Client, b blobChecker) []string {
	errs := b.judge(c.checks, t.Body)
	if c.links != nil {
		errs = append(errs, c.links(t, gh, b)...)
	}
	return errs
}

// react applies the declared after-effect, if any.
func (c ghCommand) react(gh aftereffect.Client, t target, errs []string) error {
	if c.effect == nil {
		return nil
	}
	return c.effect(gh, t, errs)
}

// exitCode is 1 when a target failed and the command fails on findings.
func (c ghCommand) exitCode(name string, failed bool) int {
	if failed && c.failOnFindings {
		fmt.Fprintf(os.Stderr, "fitness %s: one or more targets have violations.\n", name)
		return 1
	}
	return 0
}

// ghFailure reports an error that stopped a GitHub command.
func ghFailure(name string, err error) int {
	report.Error("fitness "+name, err)
	return 1
}

// prTargets is the triggering PR, or every open PR on workflow_dispatch.
func prTargets(ev ghEvent, gh aftereffect.Client, _ blobChecker) ([]target, error) {
	if ev.name == "workflow_dispatch" {
		return gh.OpenPRs()
	}
	return eventTarget(ev.name, "pull_request", ev.PullRequest)
}

// eventTarget is the one target an event carries.
func eventTarget(event, field string, t *target) ([]target, error) {
	if t == nil {
		return nil, fmt.Errorf("no %s in the %q event", field, event)
	}
	return []target{*t}, nil
}

// closeTargets is the issue an `issues: closed` event carries, unless it was
// closed as dropped work, where unfinished items are expected.
func closeTargets(ev ghEvent, _ aftereffect.Client, _ blobChecker) ([]target, error) {
	if ev.Issue == nil || ev.Action != "closed" {
		return nil, fmt.Errorf("no closed issue in the %q event", ev.name)
	}
	t := *ev.Issue
	if t.StateReason == "not_planned" || t.StateReason == "duplicate" {
		fmt.Printf("#%d closed as %s; unfinished items are allowed.\n", t.Number, t.StateReason)
		return nil, nil
	}
	t.ClosedBy = ev.Sender.Login
	return []target{t}, nil
}

// ghEvent is the part of the Actions event payload the GitHub commands read.
type ghEvent struct {
	name        string
	Action      string        `json:"action"`
	PullRequest *target       `json:"pull_request"`
	Issue       *target       `json:"issue"`
	Comment     *eventComment `json:"comment"`
	Sender      struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// readEvent loads the triggering event from the Actions environment.
func readEvent() (ghEvent, error) {
	ev := ghEvent{name: os.Getenv("GITHUB_EVENT_NAME")}
	data, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		return ev, fmt.Errorf("read the event payload: %w", err)
	}
	return ev, json.Unmarshal(data, &ev)
}
