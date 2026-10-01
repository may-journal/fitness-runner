package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

// runPlanCheck validates the triggering Plan issue, or every open Plan on
// workflow_dispatch, comments the result once per body version, and labels it. The plan
// check workflows call this in one step.
func runPlanCheck() int {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ev, err := readEvent()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	gh, err := newGHClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	return planCheck(ev, gh, newBodyChecker(root))
}

// planCheck is runPlanCheck past the environment: it validates each target
// Plan, keeps one live verdict comment, and labels the verdict.
func planCheck(ev ghEvent, gh githubAPI, c bodyChecker) int {
	targets, err := planTargets(ev, gh)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	failed, err := checkPlans(targets, gh, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness plan-check: "+err.Error())
		return 1
	}
	if failed {
		fmt.Fprintln(os.Stderr, "One or more plan issues have violations.")
		return 1
	}
	return 0
}

// checkPlans validates, comments on, and labels each target, reporting
// whether any failed.
func checkPlans(targets []target, gh githubAPI, c bodyChecker) (failed bool, err error) {
	for _, t := range targets {
		errs := validatePlan(t, c)
		failed = failed || len(errs) > 0
		if err := postVerdict(t, errs, gh); err != nil {
			return failed, err
		}
		if err := labelVerdict(t.Number, len(errs) == 0, gh); err != nil {
			return failed, err
		}
	}
	return failed, nil
}

// planTargets is the triggering issue, or every open Plan on
// workflow_dispatch.
func planTargets(ev ghEvent, gh githubAPI) ([]target, error) {
	if ev.name == "workflow_dispatch" {
		return gh.openPlans()
	}
	if ev.Issue == nil {
		return nil, fmt.Errorf("no issue in the %q event", ev.name)
	}
	return []target{*ev.Issue}, nil
}

// validatePlan runs the plan-structure check and the prose checks on a body.
func validatePlan(t target, c bodyChecker) []string {
	errs := c.errorsOf("plan-structure", nil, t.Body)
	bodyFile, err := writeBody(t.Body)
	if err != nil {
		return append(errs, err.Error())
	}
	defer os.Remove(bodyFile)
	return append(errs, c.docErrors(bodyFile)...)
}

// postVerdict comments the result once per body version. A pass that follows
// a pass edits that comment in place with a timestamp; anything else posts a
// new one. Every older verdict is then hidden as outdated.
func postVerdict(t target, errs []string, gh githubAPI) error {
	mark := planMarker(t.Body)
	prior, err := gh.verdicts(t.Number)
	if err != nil {
		return err
	}
	keep, err := liveVerdict(t.Number, errs, mark, prior, gh)
	if err != nil {
		return err
	}
	return hideOlder(prior, keep, gh)
}

// liveVerdict returns the index of the verdict judging this body version,
// writing it first when none does; -1 means a new comment was posted.
func liveVerdict(n int, errs []string, mark string, prior []verdict, gh githubAPI) (int, error) {
	if i := indexOfMarker(prior, mark); i >= 0 {
		return i, nil
	}
	return writeVerdict(n, errs, mark, prior, gh)
}

// hideOlder hides every verdict but the live one, skipping those already
// hidden.
func hideOlder(prior []verdict, keep int, gh githubAPI) error {
	for i, v := range prior {
		if i == keep || v.IsHidden {
			continue
		}
		if err := gh.hideComment(v.NodeID); err != nil {
			return err
		}
	}
	return nil
}

// indexOfMarker is the verdict already judging this body version, or -1.
func indexOfMarker(prior []verdict, mark string) int {
	for i, v := range prior {
		if strings.Contains(v.Body, mark) {
			return i
		}
	}
	return -1
}

// writeVerdict edits the newest verdict when it and this one both pass, and
// otherwise posts a new comment. It returns the index of the edited verdict,
// or -1 when it posted.
func writeVerdict(n int, errs []string, mark string, prior []verdict, gh githubAPI) (int, error) {
	last := len(prior) - 1
	if len(errs) == 0 && last >= 0 && strings.HasPrefix(prior[last].Body, passIcon) {
		return last, gh.editComment(prior[last].ID, validatedComment(mark))
	}
	return -1, gh.comment(n, planComment(errs, mark))
}

// Labels plan-check keeps on each checked Issue.
const (
	fitnessLabel = "fitness"
	validLabel   = "fitness-valid"
	invalidLabel = "fitness-invalid"
)

// labelColors are the colors a missing label is created with.
var labelColors = map[string]string{fitnessLabel: "5319e7", validLabel: "0e8a16", invalidLabel: "d93f0b"}

// labelVerdict adds the fitness label and the label matching the verdict,
// and removes the opposite verdict label.
func labelVerdict(n int, valid bool, gh githubAPI) error {
	want, drop := verdictLabels(valid)
	_, labels, err := gh.issue(n)
	if err != nil {
		return err
	}
	if err := addMissing(n, labels, []string{fitnessLabel, want}, gh); err != nil {
		return err
	}
	if hasLabel(labels, drop) {
		return gh.removeLabel(n, drop)
	}
	return nil
}

// verdictLabels is the label to add and the one to remove for a verdict.
func verdictLabels(valid bool) (want, drop string) {
	if valid {
		return validLabel, invalidLabel
	}
	return invalidLabel, validLabel
}

// addMissing adds the wanted labels the Issue lacks, creating each in the
// repo first when it does not exist.
func addMissing(n int, labels, want []string, gh githubAPI) error {
	add := missingLabels(labels, want)
	for _, l := range add {
		if err := gh.ensureLabel(l, labelColors[l]); err != nil {
			return err
		}
	}
	if len(add) == 0 {
		return nil
	}
	return gh.addLabels(n, add)
}

// missingLabels is want minus labels.
func missingLabels(labels, want []string) []string {
	var add []string
	for _, l := range want {
		if !hasLabel(labels, l) {
			add = append(add, l)
		}
	}
	return add
}

// planMarker tags a comment with a hash of the body it judged; the format
// matches the earlier JavaScript, so existing comments still count.
func planMarker(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "<!-- fitness:plan-structure:" + hex.EncodeToString(sum[:])[:12] + " -->"
}

// passIcon starts every passing verdict.
const passIcon = "✅"

// now is the clock for the validated timestamp; tests pin it.
var now = time.Now

// validatedComment replaces a passing verdict when a later body passes too.
func validatedComment(mark string) string {
	return passIcon + " Validated (updated " + now().UTC().Format("2006-01-02 15:04 UTC") + ")" + footer(mark)
}

// planComment is the verdict comment for a Plan body.
func planComment(errs []string, mark string) string {
	if len(errs) == 0 {
		return passIcon + " Plan looks good." + footer(mark)
	}
	return "❌ Plan needs work:\n" + strings.TrimSuffix(bulletList(errs), "\n") + footer(mark)
}

// footer ends a verdict with its attribution and the body-version marker.
func footer(mark string) string {
	return "\n\n" + attribution() + "\n\n" + mark
}
