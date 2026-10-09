package main

import (
	"encoding/json"
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
	targets func(ev ghEvent, gh aftereffect.Client) ([]target, error)
	checks  []string
	// links judges what a body points at, such as the issues a PR closes.
	links  func(t target, gh aftereffect.Client, b blobChecker) []string
	effect aftereffect.Effect
	// failOnFindings exits 1 when any target fails; close-check instead
	// succeeds once its effect has reopened the issue.
	failOnFindings bool
}

// ghCommands are the GitHub commands, by subcommand name.
var ghCommands = map[string]ghCommand{
	"plan-check": {
		kind: "issues", targets: planTargets,
		checks: append([]string{"plan-structure"}, docChecks...),
		effect: aftereffect.PlanVerdict, failOnFindings: true,
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
}

// runGH runs the named GitHub command against the Actions event.
func runGH(name string) int {
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
	targets, err := c.targets(ev, gh)
	if err != nil {
		return ghFailure(name, err)
	}
	rep := newTargetReport(name, c.kind)
	failed := false
	for _, t := range targets {
		errs := c.judge(t, gh, b)
		rep.add(t.Number, errs)
		failed = failed || len(errs) > 0
		if err := c.react(gh, t, errs); err != nil {
			rep.finish()
			return ghFailure(fmt.Sprintf("%s #%d", name, t.Number), err)
		}
	}
	rep.finish()
	return c.exitCode(name, failed)
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

// planTargets is the triggering issue, or every open Plan on
// workflow_dispatch.
func planTargets(ev ghEvent, gh aftereffect.Client) ([]target, error) {
	if ev.name == "workflow_dispatch" {
		return gh.OpenPlans()
	}
	return eventTarget(ev.name, "issue", ev.Issue)
}

// prTargets is the triggering PR, or every open PR on workflow_dispatch.
func prTargets(ev ghEvent, gh aftereffect.Client) ([]target, error) {
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
func closeTargets(ev ghEvent, _ aftereffect.Client) ([]target, error) {
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
	Action      string  `json:"action"`
	PullRequest *target `json:"pull_request"`
	Issue       *target `json:"issue"`
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
